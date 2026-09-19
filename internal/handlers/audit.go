package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/numeric"
	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

// AuditDB exposes the historical inputs needed to audit the H1 hypothesis.
type AuditDB interface {
	GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error)
	GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error)
	GetWorkloadItemSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error)
}

// AuditResponse compares the reported projected_monthly_waste_usd against a
// freshly recomputed reference value derived from the same raw inputs and the
// configured cost model. Designed to validate H1 (error < 10%).
type AuditResponse struct {
	AuditType           string       `json:"audit_type"`
	Window              string       `json:"window"`
	StartAt             string       `json:"start_at"`
	EndAt               string       `json:"end_at"`
	ScrapesCount        int          `json:"scrapes_count"`
	AuditedScrapesCount int          `json:"audited_scrapes_count"`
	SkippedScrapesCount int          `json:"skipped_scrapes_count"`
	Conclusive          bool         `json:"conclusive"`
	CostModel           string       `json:"cost_model"`
	Reported            float64      `json:"reported_projected_monthly_waste_usd"`
	Recalculated        float64      `json:"recalculated_projected_monthly_waste_usd"`
	ReportedCPU         float64      `json:"reported_cpu_projected_monthly_waste_usd"`
	RecalculatedCPU     float64      `json:"recalculated_cpu_projected_monthly_waste_usd"`
	ReportedMemory      float64      `json:"reported_memory_projected_monthly_waste_usd"`
	RecalculatedMemory  float64      `json:"recalculated_memory_projected_monthly_waste_usd"`
	AbsoluteErrorUSD    float64      `json:"absolute_error_usd"`
	ErrorPercent        float64      `json:"error_percent"`
	WithinTolerance     bool         `json:"within_10pct_tolerance"`
	Points              []AuditPoint `json:"points,omitempty"`
}

type AuditPoint struct {
	ScrapeID           int     `json:"scrape_id"`
	Reported           float64 `json:"reported"`
	Recalculated       float64 `json:"recalculated"`
	ReportedCPU        float64 `json:"reported_cpu"`
	RecalculatedCPU    float64 `json:"recalculated_cpu"`
	ReportedMemory     float64 `json:"reported_memory"`
	RecalculatedMemory float64 `json:"recalculated_memory"`
	AbsoluteError      float64 `json:"absolute_error"`
	ErrorPercent       float64 `json:"error_percent"`
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
// @Description  Recalcula os custos de CPU e memória com implementação independente e verifica apenas a consistência interna do valor reportado; não valida faturamento real.
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
			AuditType: "internal_consistency",
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

	// Fetch per-item snapshots for re-aggregation.
	itemSnapshots, err := h.db.GetWorkloadItemSnapshotsForScrapes(ctx, scrapeIDs)
	if err != nil {
		h.logger.Error("audit: failed to load item snapshots", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load workload item snapshots")
		return
	}

	// Group items by ScrapeID to allow independent re-aggregation.
	itemsByScrape := make(map[int][]*entities.WorkloadItemSnapshotModel)
	for _, item := range itemSnapshots {
		itemsByScrape[item.ScrapeID()] = append(itemsByScrape[item.ScrapeID()], item)
	}

	includeDetails := q.Get("details") == "true"
	var totalReported, totalRecalc, reportedCPU, recalcCPU, reportedMemory, recalcMemory float64
	audited := 0
	points := make([]AuditPoint, 0, len(indicators))
	for _, ind := range indicators {
		// Methodological Guard: Only process scrapes that have both global and per-item data.
		items, ok := itemsByScrape[ind.ScrapeID()]
		if !ok {
			h.logger.Warn("audit: scrape has global indicator but no item snapshots", "scrape_id", ind.ScrapeID())
			continue
		}

		reported := ind.ProjectedMonthlyWasteUSD()
		var scrapeCPU, scrapeMemory float64
		for _, item := range items {
			scrapeCPU += auditCPUCost(item.CPURequestedCores(), item.CPUUsedCores(), h.cpuHourlyUSD, h.monthlyHours)
			scrapeMemory += auditMemoryCost(item.MemoryRequestedBytes(), item.MemoryUsedBytes(), h.memoryGiBHourlyUSD, h.monthlyHours)
		}
		scrapeCPU = numeric.Round4(scrapeCPU)
		scrapeMemory = numeric.Round4(scrapeMemory)
		scrapeRecalc := numeric.Round4(scrapeCPU + scrapeMemory)
		audited++

		totalReported += reported
		totalRecalc += scrapeRecalc
		reportedCPU += ind.CPUProjectedMonthlyWasteUSD()
		reportedMemory += ind.MemoryProjectedMonthlyWasteUSD()
		recalcCPU += scrapeCPU
		recalcMemory += scrapeMemory
		if includeDetails {
			abs := math.Abs(reported - scrapeRecalc)
			pct := auditErrorPercent(reported, scrapeRecalc)
			points = append(points, AuditPoint{
				ScrapeID:           ind.ScrapeID(),
				Reported:           reported,
				Recalculated:       scrapeRecalc,
				ReportedCPU:        ind.CPUProjectedMonthlyWasteUSD(),
				RecalculatedCPU:    scrapeCPU,
				ReportedMemory:     ind.MemoryProjectedMonthlyWasteUSD(),
				RecalculatedMemory: scrapeMemory,
				AbsoluteError:      numeric.Round4(abs),
				ErrorPercent:       pct,
			})
		}
	}

	abs := math.Abs(totalReported - totalRecalc)
	pct := auditErrorPercent(totalReported, totalRecalc)
	conclusive := audited > 0

	resp := AuditResponse{
		AuditType:           "internal_consistency",
		Window:              timeWindowString(*startTime, *endTime),
		StartAt:             startTime.Format(time.RFC3339),
		EndAt:               endTime.Format(time.RFC3339),
		ScrapesCount:        len(scrapes),
		AuditedScrapesCount: audited,
		SkippedScrapesCount: len(scrapes) - audited,
		Conclusive:          conclusive,
		CostModel:           h.costModelLabel,
		Reported:            numeric.Round4(totalReported),
		Recalculated:        numeric.Round4(totalRecalc),
		ReportedCPU:         numeric.Round4(reportedCPU),
		RecalculatedCPU:     numeric.Round4(recalcCPU),
		ReportedMemory:      numeric.Round4(reportedMemory),
		RecalculatedMemory:  numeric.Round4(recalcMemory),
		AbsoluteErrorUSD:    numeric.Round4(abs),
		ErrorPercent:        pct,
		WithinTolerance:     conclusive && pct < 10.0,
		Points:              points,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func auditCPUCost(requested, used, hourlyUSD, monthlyHours float64) float64 {
	waste := requested - used
	if waste < 0 {
		waste = 0
	}
	return numeric.Round4(waste * hourlyUSD * monthlyHours)
}

func auditMemoryCost(requested, used, hourlyUSD, monthlyHours float64) float64 {
	waste := requested - used
	if waste < 0 {
		waste = 0
	}
	return numeric.Round4((waste / (1024 * 1024 * 1024)) * hourlyUSD * monthlyHours)
}

func auditErrorPercent(reported, reference float64) float64 {
	if reference == 0 {
		if reported == 0 {
			return 0
		}
		return 100
	}
	return numeric.Round4(math.Abs(reported-reference) / math.Abs(reference) * 100)
}

func timeWindowString(start, end time.Time) string {
	d := end.Sub(start)
	return d.Truncate(time.Second).String()
}
