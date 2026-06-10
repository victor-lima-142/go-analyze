package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-analyze/internal/metrics"
)

type WorkloadDetailsResponse struct {
	Namespace  string                     `json:"namespace"`
	Pod        string                     `json:"pod"`
	Container  string                     `json:"container"`
	Current    *CurrentWorkloadMetrics    `json:"current"`
	Historical *HistoricalWorkloadSummary `json:"historical"`
}

type CurrentWorkloadMetrics struct {
	CPURequestedCores        float64  `json:"cpu_requested_cores"`
	CPUUsedCores             float64  `json:"cpu_used_cores"`
	CPULimitCores            *float64 `json:"cpu_limit_cores,omitempty"`
	CPUWasteRatio            float64  `json:"cpu_waste_ratio"`
	MemoryRequestedBytes     float64  `json:"memory_requested_bytes"`
	MemoryUsedBytes          float64  `json:"memory_used_bytes"`
	MemoryLimitBytes         *float64 `json:"memory_limit_bytes,omitempty"`
	MemWasteRatio            float64  `json:"mem_waste_ratio"`
	OOMRiskScore             float64  `json:"oom_risk_score"`
	ProjectedMonthlyWasteUSD float64  `json:"projected_monthly_waste_usd"`
	Timestamp                string   `json:"timestamp"`
}

type HistoricalWorkloadSummary struct {
	Last10ConsolidationsCount   int                     `json:"last_10_consolidations_count"`
	AvgCPURequestedCores        float64                 `json:"avg_cpu_requested_cores"`
	AvgCPUUsedCores             float64                 `json:"avg_cpu_used_cores"`
	AvgCPUWasteRatio            float64                 `json:"avg_cpu_waste_ratio"`
	AvgMemoryRequestedBytes     float64                 `json:"avg_memory_requested_bytes"`
	AvgMemoryUsedBytes          float64                 `json:"avg_memory_used_bytes"`
	AvgMemWasteRatio            float64                 `json:"avg_mem_waste_ratio"`
	AvgProjectedMonthlyWasteUSD float64                 `json:"avg_projected_monthly_waste_usd"`
	Points                      []HistoricalDetailPoint `json:"points"`
}

type HistoricalDetailPoint struct {
	ConsolidationID          int      `json:"consolidation_id"`
	Timestamp                string   `json:"timestamp"`
	CPURequestedCores        float64  `json:"cpu_requested_cores"`
	CPUUsedCores             float64  `json:"cpu_used_cores"`
	CPULimitCores            *float64 `json:"cpu_limit_cores,omitempty"`
	CPUWasteRatio            float64  `json:"cpu_waste_ratio"`
	MemoryRequestedBytes     float64  `json:"memory_requested_bytes"`
	MemoryUsedBytes          float64  `json:"memory_used_bytes"`
	MemoryLimitBytes         *float64 `json:"memory_limit_bytes,omitempty"`
	MemWasteRatio            float64  `json:"mem_waste_ratio"`
	OOMRiskScore             float64  `json:"oom_risk_score"`
	ProjectedMonthlyWasteUSD float64  `json:"projected_monthly_waste_usd"`
}

type WorkloadDetailsHandler struct {
	calculator    Calculator
	store         ConsolidationStore
	defaultWindow string
	logger        *slog.Logger
}

func NewWorkloadDetailsHandler(calculator Calculator, store ConsolidationStore, defaultWindow string, logger *slog.Logger) *WorkloadDetailsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkloadDetailsHandler{calculator: calculator, store: store, defaultWindow: defaultWindow, logger: logger}
}

// ServeHTTP handles GET /api/v1/workloads/details
// @Summary      Obter detalhes de um workload específico
// @Description  Retorna as métricas atuais de execução em tempo real e um histórico das consolidações registradas para o workload. O parâmetro `pod` é tratado como prefixo (LIKE) quando não contém o caractere %.
// @Tags         workloads
// @Accept       json
// @Produce      json
// @Param        namespace query     string  true   "Namespace do Kubernetes"
// @Param        container query     string  true   "Nome do Container no Kubernetes"
// @Param        pod       query     string  false  "Prefixo do Pod no Kubernetes (LIKE)"
// @Param        window    query     string  false  "Janela de tempo para o cálculo atual"
// @Param        period    query     string  false  "Período rápido para busca histórica (ex: 1hr, 5d, 10d)"
// @Param        start_at  query     string  false  "Data inicial para histórico (ex: RFC3339)"
// @Param        end_at    query     string  false  "Data final para histórico"
// @Param        limit     query     int     false  "Número máximo de pontos históricos a retornar (padrão: 10)"
// @Success      200       {object}  WorkloadDetailsResponse
// @Failure      400       {object}  map[string]any "Namespace e container são parâmetros obrigatórios ou datas inválidas"
// @Failure      405       {object}  map[string]any "Método HTTP não permitido"
// @Failure      500       {object}  map[string]any "Erro ao carregar os dados históricos"
// @Router       /api/v1/workloads/details [get]
func (h *WorkloadDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	namespace := q.Get("namespace")
	pod := q.Get("pod")
	container := q.Get("container")
	window := q.Get("window")

	if namespace == "" || container == "" {
		writeError(w, http.StatusBadRequest, "namespace and container are required parameters")
		return
	}

	if window == "" {
		window = h.defaultWindow
	}

	ctx := r.Context()

	var current *CurrentWorkloadMetrics
	calcRes, err := h.calculator.Calculate(ctx, metrics.QueryParams{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		Window:    window,
	})

	if err == nil && calcRes != nil {
		for _, item := range calcRes.Items {
			if item.Namespace == namespace && item.Container == container && (pod == "" || item.Pod == pod) {
				var cpuLimit *float64
				if item.CPULimitCores > 0 {
					val := item.CPULimitCores
					cpuLimit = &val
				}
				var memLimit *float64
				if item.MemoryLimitBytes > 0 {
					val := item.MemoryLimitBytes
					memLimit = &val
				}
				current = &CurrentWorkloadMetrics{
					CPURequestedCores:        item.CPURequestedCores,
					CPUUsedCores:             item.CPUUsedCores,
					CPULimitCores:            cpuLimit,
					CPUWasteRatio:            item.CPUWasteRatio,
					MemoryRequestedBytes:     item.MemoryRequestedBytes,
					MemoryUsedBytes:          item.MemoryUsedBytes,
					MemoryLimitBytes:         memLimit,
					MemWasteRatio:            item.MemWasteRatio,
					OOMRiskScore:             item.OOMRiskScore,
					ProjectedMonthlyWasteUSD: item.ProjectedMonthlyWasteUSD,
					Timestamp:                time.Now().Format(time.RFC3339),
				}
				break
			}
		}
	} else if err != nil {
		h.logger.Warn("live calculator error for details query", "error", err)
	}

	startTime, endTime, err := ParseDateFilters(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	limit := 10
	if (startTime != nil || endTime != nil) && q.Get("limit") == "" {
		limit = 0
	} else if limitStr := q.Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxPageSize {
				limit = maxPageSize
			}
		}
	}

	snapshots, err := h.store.GetLastWorkloadSnapshots(ctx, namespace, pod, container, limit, startTime, endTime)
	if err != nil {
		h.logger.Error("error fetching last workload snapshots", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load historical snapshots")
		return
	}

	var sumCpuReq, sumCpuUsed, sumCpuWaste float64
	var sumMemReq, sumMemUsed, sumMemWaste float64
	var sumProjected float64
	count := len(snapshots)

	points := make([]HistoricalDetailPoint, 0, count)
	for _, s := range snapshots {
		snap := s.Snapshot
		pt := HistoricalDetailPoint{
			ConsolidationID:          snap.ConsolidationID(),
			Timestamp:                s.ConsolidatedAt.Format(time.RFC3339),
			CPURequestedCores:        snap.CPURequestedCores(),
			CPUUsedCores:             snap.CPUUsedCores(),
			CPULimitCores:            snap.CPULimitCores(),
			CPUWasteRatio:            snap.CPUWasteRatio(),
			MemoryRequestedBytes:     snap.MemoryRequestedBytes(),
			MemoryUsedBytes:          snap.MemoryUsedBytes(),
			MemoryLimitBytes:         snap.MemoryLimitBytes(),
			MemWasteRatio:            snap.MemWasteRatio(),
			OOMRiskScore:             snap.OOMRiskScore(),
			ProjectedMonthlyWasteUSD: snap.ProjectedMonthlyWasteUSD(),
		}
		points = append(points, pt)

		sumCpuReq += snap.CPURequestedCores()
		sumCpuUsed += snap.CPUUsedCores()
		sumCpuWaste += snap.CPUWasteRatio()
		sumMemReq += snap.MemoryRequestedBytes()
		sumMemUsed += snap.MemoryUsedBytes()
		sumMemWaste += snap.MemWasteRatio()
		sumProjected += snap.ProjectedMonthlyWasteUSD()
	}

	var hist HistoricalWorkloadSummary
	hist.Last10ConsolidationsCount = count
	hist.Points = points
	if count > 0 {
		hist.AvgCPURequestedCores = sumCpuReq / float64(count)
		hist.AvgCPUUsedCores = sumCpuUsed / float64(count)
		hist.AvgCPUWasteRatio = sumCpuWaste / float64(count)
		hist.AvgMemoryRequestedBytes = sumMemReq / float64(count)
		hist.AvgMemoryUsedBytes = sumMemUsed / float64(count)
		hist.AvgMemWasteRatio = sumMemWaste / float64(count)
		hist.AvgProjectedMonthlyWasteUSD = sumProjected / float64(count)
	}

	resp := WorkloadDetailsResponse{
		Namespace:  namespace,
		Pod:        pod,
		Container:  container,
		Current:    current,
		Historical: &hist,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("error encoding details response", "error", err)
	}
}
