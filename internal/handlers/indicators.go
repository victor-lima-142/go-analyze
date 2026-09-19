package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/metrics"
)

type PaginatedIndicatorResponse struct {
	Timestamp   string                 `json:"timestamp"`
	Filters     metrics.QueryParams    `json:"filters"`
	Indicators  metrics.Indicators     `json:"indicators"`
	Inputs      metrics.Inputs         `json:"inputs"`
	CostModel   string                 `json:"cost_model,omitempty"`
	Content     []metrics.WorkloadItem `json:"content"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"pageSize"`
	TotalPages  int                    `json:"totalPages"`
	TotalItems  int                    `json:"totalItems"`
	HasNext     bool                   `json:"hasNext"`
	HasPrevious bool                   `json:"hasPrevious"`
}

type IndicatorHandler struct {
	calculator    Calculator
	defaultWindow string
	logger        *slog.Logger
}

func NewIndicatorHandler(calculator Calculator, defaultWindow string, logger *slog.Logger) *IndicatorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &IndicatorHandler{calculator: calculator, defaultWindow: defaultWindow, logger: logger}
}

// ServeHTTP handles GET /api/v1/indicators
// @Summary      Obter indicadores calculados atuais
// @Description  Calcula e retorna os indicadores atuais de otimização de recursos a partir do Prometheus.
// @Tags         indicators
// @Accept       json
// @Produce      json
// @Param        window    query     string  false  "Janela de tempo para o cálculo (ex: 30s, 1m, 5m, 10m, 30m, 1h)"
// @Param        page      query     int     false  "Número da página (padrão: 1)"
// @Param        pageSize  query     int     false  "Tamanho da página (padrão: 50, máx: 1000)"
// @Param        limit     query     int     false  "Limite alternativo de tamanho de página"
// @Param        namespace query     string  false  "Filtrar por Namespace do Kubernetes"
// @Param        pod       query     string  false  "Filtrar por Pod do Kubernetes"
// @Param        container query     string  false  "Filtrar por Container do Kubernetes"
// @Success      200       {object}  PaginatedIndicatorResponse
// @Failure      405       {object}  map[string]any "Método HTTP não permitido"
// @Failure      502       {object}  map[string]any "Falha ao consultar o Prometheus ou calcular indicadores"
// @Router       /api/v1/indicators [get]
func (h *IndicatorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	window := q.Get("window")
	if window == "" {
		window = h.defaultWindow
	}

	page, pageSize := clampPagination(q)

	params := metrics.QueryParams{
		Namespace: q.Get("namespace"),
		Pod:       q.Get("pod"),
		Container: q.Get("container"),
		Window:    metrics.NormalizeWindow(window),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.calculator.Calculate(ctx, params)
	if err != nil {
		h.logger.Error("indicator calculation failed", "error", err, "filters", params)
		writeError(w, http.StatusBadGateway, "failed to query prometheus or calculate indicators")
		return
	}

	totalItems := len(result.Items)
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize

	var content []metrics.WorkloadItem
	if start < totalItems {
		if end > totalItems {
			end = totalItems
		}
		content = result.Items[start:end]
	} else {
		content = []metrics.WorkloadItem{}
	}

	paginatedResp := PaginatedIndicatorResponse{
		Timestamp:   time.Now().Format(time.RFC3339),
		Filters:     result.Filters,
		Indicators:  result.Indicators,
		Inputs:      result.Inputs,
		CostModel:   result.CostModel,
		Content:     content,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(paginatedResp)
}
