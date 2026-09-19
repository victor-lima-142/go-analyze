package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func NewConsolidatedHandler(store ConsolidationStore, logger *slog.Logger) *ConsolidatedHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConsolidatedHandler{store: store, logger: logger}
}

// ServeHTTP handles GET /api/v1/consolidated
// @Summary      Listar consolidações históricas
// @Description  Retorna uma lista paginada das consolidações salvas no banco de dados, com filtros de data opcionais e opção de incluir workloads detalhados.
// @Tags         consolidated
// @Accept       json
// @Produce      json
// @Param        page          query     int     false  "Número da página (padrão: 1)"
// @Param        pageSize      query     int     false  "Tamanho da página (padrão: 50, máx: 1000)"
// @Param        limit         query     int     false  "Limite alternativo de tamanho de página"
// @Param        include_items query     bool    false  "Se deve incluir itens consolidados de workloads (padrão: true)"
// @Param        period        query     string  false  "Período rápido (ex: 1min, 5min, 10min, 30min, 1hr, 5d, 10d)"
// @Param        start_at      query     string  false  "Data inicial (ex: RFC3339 ou 2006-01-02)"
// @Param        end_at        query     string  false  "Data final"
// @Success      200           {object}  PaginatedConsolidatedResponse
// @Failure      400           {object}  map[string]any "Parâmetros de filtro ou data inválidos"
// @Failure      405           {object}  map[string]any "Método HTTP não permitido"
// @Failure      500           {object}  map[string]any "Erro interno ao consultar o banco de dados"
// @Router       /api/v1/consolidated [get]
func (h *ConsolidatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	page, pageSize := clampPagination(q)
	includeItems := q.Get("include_items") != "false"

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	startTime, endTime, err := ParseDateFilters(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	offset := (page - 1) * pageSize
	consolidations, totalItems, err := h.store.GetConsolidations(ctx, pageSize, offset, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to retrieve consolidations", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query consolidations from database")
		return
	}

	responseList := make([]ConsolidatedResponse, len(consolidations))
	for i, c := range consolidations {
		resp := ConsolidatedResponse{
			ID:             c.ID(),
			ConsolidatedAt: c.ConsolidatedAt().Format(time.RFC3339),
			Timestamp:      c.ConsolidatedAt().Format(time.RFC3339),
			StartTime:      c.StartTime().Format(time.RFC3339),
			EndTime:        c.EndTime().Format(time.RFC3339),
			ScrapesCount:   c.ScrapesCount(),
			Indicators: ConsolidatedInd{
				CPUWasteRatio:                  c.CPUWasteRatio(),
				MemWasteRatio:                  c.MemWasteRatio(),
				CPUProjectedMonthlyWasteUSD:    c.CPUProjectedMonthlyWasteUSD(),
				MemoryProjectedMonthlyWasteUSD: c.MemoryProjectedMonthlyWasteUSD(),
				ProjectedMonthlyWasteUSD:       c.ProjectedMonthlyWasteUSD(),
			},
			Inputs: ConsolidatedInputs{
				TotalCPURequested:    c.TotalCPURequested(),
				TotalCPUUsed:         c.TotalCPUUsed(),
				TotalMemoryRequested: c.TotalMemoryRequested(),
				TotalMemoryUsed:      c.TotalMemoryUsed(),
			},
		}

		if includeItems {
			wlSnapshots, err := h.store.GetConsolidatedWorkloads(ctx, c.ID())
			if err != nil {
				h.logger.Error("failed to retrieve consolidated workloads", "consolidation_id", c.ID(), "error", err)
				writeError(w, http.StatusInternalServerError, "failed to query workload items from database")
				return
			}
			items := make([]ConsolidatedItem, len(wlSnapshots))
			for j, wl := range wlSnapshots {
				items[j] = ConsolidatedItem{
					Namespace:                      wl.Namespace(),
					Pod:                            wl.Pod(),
					Container:                      wl.Container(),
					CPURequestedCores:              wl.CPURequestedCores(),
					CPUUsedCores:                   wl.CPUUsedCores(),
					CPULimitCores:                  wl.CPULimitCores(),
					MemoryRequestedBytes:           wl.MemoryRequestedBytes(),
					MemoryUsedBytes:                wl.MemoryUsedBytes(),
					MemoryLimitBytes:               wl.MemoryLimitBytes(),
					CPUWasteRatio:                  wl.CPUWasteRatio(),
					MemWasteRatio:                  wl.MemWasteRatio(),
					CPUProjectedMonthlyWasteUSD:    wl.CPUProjectedMonthlyWasteUSD(),
					MemoryProjectedMonthlyWasteUSD: wl.MemoryProjectedMonthlyWasteUSD(),
					ProjectedMonthlyWasteUSD:       wl.ProjectedMonthlyWasteUSD(),
				}
			}
			resp.Items = items
		}

		responseList[i] = resp
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}

	paginatedResp := PaginatedConsolidatedResponse{
		Content:     responseList,
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
