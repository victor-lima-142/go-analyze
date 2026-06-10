package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/metrics"
	"github.com/victor-lima-142/go-analyze/internal/numeric"
	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

// AuditDB exposes the historical inputs needed to audit the H1 hypothesis.
type AuditDB interface {
	GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error)
	GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error)
}

// AuditResponse compares the reported projected_monthly_waste_usd against a
// freshly recomputed reference value derived from the same raw inputs and the
// configured cost model. Designed to validate H1 (error < 10%).
type AuditResponse struct {
	Window           string       `json:"window"`
	StartAt          string       `json:"start_at"`
	EndAt            string       `json:"end_at"`
	ScrapesCount     int          `json:"scrapes_count"`
	CostModel        string       `json:"cost_model"`
	Reported         float64      `json:"reported_projected_monthly_waste_usd"`
	Recalculated     float64      `json:"recalculated_projected_monthly_waste_usd"`
	AbsoluteErrorUSD float64      `json:"absolute_error_usd"`
	ErrorPercent     float64      `json:"error_percent"`
	WithinTolerance  bool         `json:"within_10pct_tolerance"`
	Points           []AuditPoint `json:"points,omitempty"`
}

type AuditPoint struct {
	ScrapeID      int     `json:"scrape_id"`
	Reported      float64 `json:"reported"`
	Recalculated  float64 `json:"recalculated"`
	AbsoluteError float64 `json:"absolute_error"`
	ErrorPercent  float64 `json:"error_percent"`
}

type AuditHandler struct {
	db                 AuditDB
	cpuHourlyUSD       float64
	memoryGiBHourlyUSD float64
	monthlyHours       float64
	costModelLabel     string
	logger             *slog.Logger
}

func NewAuditHandler(db AuditDB, cpuHourlyUSD, memoryGiBHourlyUSD, monthlyHours float64, costModelLabel string, logger *slog.Logger) *AuditHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditHandler{
		db:                 db,
		cpuHourlyUSD:       cpuHourlyUSD,
		memoryGiBHourlyUSD: memoryGiBHourlyUSD,
		monthlyHours:       monthlyHours,
		costModelLabel:     costModelLabel,
		logger:             logger,
	}
}

// ServeHTTP handles GET /api/v1/audit
// @Summary      Auditar projeção de custo
// @Description  Recalcula projected_monthly_waste_usd a partir dos inputs brutos persistidos e compara com o valor reportado, validando o critério H1 (erro < 10%) do TCC.
// @Tags         audit
// @Produce      json
// @Param        period   query  string  false  "Período rápido (ex: 1hr, 5d, 10d)"
// @Param        start_at query  string  false  "Data inicial (RFC3339)"
// @Param        end_at   query  string  false  "Data final"
// @Param        details  query  bool    false  "Inclui ponto-a-ponto"
// @Success      200      {object}  AuditResponse
// @Failure      400      {object}  map[string]any
// @Failure      500      {object}  map[string]any
// @Router       /api/v1/audit [get]
func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()

	startTime, endTime, err := ParseDateFilters(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if startTime == nil {
		t := time.Now().Add(-1 * time.Hour)
		startTime = &t
	}
	if endTime == nil {
		t := time.Now()
		endTime = &t
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	scrapes, err := h.db.GetScrapesInRange(ctx, *startTime, *endTime)
	if err != nil {
		h.logger.Error("audit: failed to load scrapes", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load scrapes")
		return
	}
	if len(scrapes) == 0 {
		_ = json.NewEncoder(w).Encode(AuditResponse{
			Window:    timeWindowString(*startTime, *endTime),
			StartAt:   startTime.Format(time.RFC3339),
			EndAt:     endTime.Format(time.RFC3339),
			CostModel: h.costModelLabel,
		})
		return
	}

	scrapeIDs := make([]int, len(scrapes))
	for i, s := range scrapes {
		scrapeIDs[i] = s.ID()
	}
	indicators, err := h.db.GetIndicatorSnapshotsForScrapes(ctx, scrapeIDs)
	if err != nil {
		h.logger.Error("audit: failed to load indicators", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load indicator snapshots")
		return
	}

	includeDetails := q.Get("details") == "true"
	var totalReported, totalRecalc float64
	points := make([]AuditPoint, 0, len(indicators))
	for _, ind := range indicators {
		reported := ind.ProjectedMonthlyWasteUSD()
		recalc := metrics.ProjectedMonthlyWasteUSD(
			ind.TotalCPURequested(), ind.TotalCPUUsed(),
			ind.TotalMemoryRequested(), ind.TotalMemoryUsed(),
			h.cpuHourlyUSD, h.memoryGiBHourlyUSD, h.monthlyHours,
		)
		totalReported += reported
		totalRecalc += recalc
		if includeDetails {
			abs := math.Abs(reported - recalc)
			pct := 0.0
			if recalc != 0 {
				pct = numeric.Round4(abs / math.Abs(recalc) * 100)
			}
			points = append(points, AuditPoint{
				ScrapeID:      ind.ScrapeID(),
				Reported:      reported,
				Recalculated:  recalc,
				AbsoluteError: numeric.Round4(abs),
				ErrorPercent:  pct,
			})
		}
	}

	abs := math.Abs(totalReported - totalRecalc)
	pct := 0.0
	if totalRecalc != 0 {
		pct = numeric.Round4(abs / math.Abs(totalRecalc) * 100)
	}

	resp := AuditResponse{
		Window:           timeWindowString(*startTime, *endTime),
		StartAt:          startTime.Format(time.RFC3339),
		EndAt:            endTime.Format(time.RFC3339),
		ScrapesCount:     len(scrapes),
		CostModel:        h.costModelLabel,
		Reported:         numeric.Round4(totalReported),
		Recalculated:     numeric.Round4(totalRecalc),
		AbsoluteErrorUSD: numeric.Round4(abs),
		ErrorPercent:     pct,
		WithinTolerance:  pct < 10.0,
		Points:           points,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func timeWindowString(start, end time.Time) string {
	d := end.Sub(start)
	return d.Truncate(time.Second).String()
}
