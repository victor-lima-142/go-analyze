package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/metrics"
	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

type Calculator interface {
	Calculate(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error)
}

type ConsolidatedResponse struct {
	ID             int                `json:"id"`
	ConsolidatedAt string             `json:"consolidated_at"`
	Timestamp      string             `json:"timestamp"`
	StartTime      string             `json:"start_time"`
	EndTime        string             `json:"end_time"`
	ScrapesCount   int                `json:"scrapes_count"`
	Indicators     ConsolidatedInd    `json:"indicators"`
	Inputs         ConsolidatedInputs `json:"inputs"`
	Items          []ConsolidatedItem `json:"items,omitempty"`
}

type ConsolidatedInd struct {
	CPUWasteRatio                  float64 `json:"cpu_waste_ratio"`
	MemWasteRatio                  float64 `json:"mem_waste_ratio"`
	CPUProjectedMonthlyWasteUSD    float64 `json:"cpu_projected_monthly_waste_usd"`
	MemoryProjectedMonthlyWasteUSD float64 `json:"memory_projected_monthly_waste_usd"`
	ProjectedMonthlyWasteUSD       float64 `json:"projected_monthly_waste_usd"`
}

type ConsolidatedInputs struct {
	TotalCPURequested    float64 `json:"total_cpu_requested"`
	TotalCPUUsed         float64 `json:"total_cpu_used"`
	TotalMemoryRequested float64 `json:"total_memory_requested_bytes"`
	TotalMemoryUsed      float64 `json:"total_memory_used_bytes"`
}

type ConsolidatedItem struct {
	Namespace                      string   `json:"namespace"`
	Pod                            string   `json:"pod"`
	Container                      string   `json:"container"`
	CPURequestedCores              float64  `json:"cpu_requested_cores"`
	CPUUsedCores                   float64  `json:"cpu_used_cores"`
	CPULimitCores                  *float64 `json:"cpu_limit_cores,omitempty"`
	MemoryRequestedBytes           float64  `json:"memory_requested_bytes"`
	MemoryUsedBytes                float64  `json:"memory_used_bytes"`
	MemoryLimitBytes               *float64 `json:"memory_limit_bytes,omitempty"`
	CPUWasteRatio                  float64  `json:"cpu_waste_ratio"`
	MemWasteRatio                  float64  `json:"mem_waste_ratio"`
	CPUProjectedMonthlyWasteUSD    float64  `json:"cpu_projected_monthly_waste_usd"`
	MemoryProjectedMonthlyWasteUSD float64  `json:"memory_projected_monthly_waste_usd"`
	ProjectedMonthlyWasteUSD       float64  `json:"projected_monthly_waste_usd"`
}

type ConsolidationStore interface {
	GetConsolidations(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error)
	GetConsolidatedWorkloads(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error)
	GetLastWorkloadSnapshots(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error)
}

type ConsolidationStoreAdapter struct {
	GetConsolidationsFunc        func(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error)
	GetConsolidatedWorkloadsFunc func(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error)
	GetLastWorkloadSnapshotsFunc func(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error)
}

func (a *ConsolidationStoreAdapter) GetConsolidations(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
	if a.GetConsolidationsFunc != nil {
		return a.GetConsolidationsFunc(ctx, limit, offset, startTime, endTime)
	}
	return nil, 0, nil
}

func (a *ConsolidationStoreAdapter) GetConsolidatedWorkloads(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error) {
	if a.GetConsolidatedWorkloadsFunc != nil {
		return a.GetConsolidatedWorkloadsFunc(ctx, consolidationID)
	}
	return nil, nil
}

func (a *ConsolidationStoreAdapter) GetLastWorkloadSnapshots(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error) {
	if a.GetLastWorkloadSnapshotsFunc != nil {
		return a.GetLastWorkloadSnapshotsFunc(ctx, namespace, pod, container, limit, startTime, endTime)
	}
	return nil, nil
}

func parseTime(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

func ParseDateFilters(q url.Values) (*time.Time, *time.Time, error) {
	period := q.Get("period")
	startAtStr := q.Get("start_at")
	endAtStr := q.Get("end_at")

	if period != "" {
		var duration time.Duration
		switch period {
		case "1min":
			duration = 1 * time.Minute
		case "5min":
			duration = 5 * time.Minute
		case "10min":
			duration = 10 * time.Minute
		case "30min":
			duration = 30 * time.Minute
		case "1hr":
			duration = 1 * time.Hour
		case "5d":
			duration = 5 * 24 * time.Hour
		case "10d":
			duration = 10 * 24 * time.Hour
		default:
			return nil, nil, fmt.Errorf("invalid period: %s. Supported values: 1min, 5min, 10min, 30min, 1hr, 5d, 10d", period)
		}
		endTime := time.Now()
		startTime := endTime.Add(-duration)
		return &startTime, &endTime, nil
	}

	var startTime, endTime *time.Time

	if startAtStr != "" {
		st, err := parseTime(startAtStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start_at format: %w", err)
		}
		startTime = &st
	}
	if endAtStr != "" {
		et, err := parseTime(endAtStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end_at format: %w", err)
		}
		endTime = &et
	}

	if startTime != nil && endTime == nil {
		now := time.Now()
		endTime = &now
	}

	return startTime, endTime, nil
}

// clampPagination validates query string pagination params and enforces a hard
// upper bound so the API cannot be coerced into expensive queries by accident.
func clampPagination(q url.Values) (page, pageSize int) {
	page = 1
	if v := q.Get("page"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize = defaultPageSize
	if v := q.Get("pageSize"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			pageSize = parsed
		}
	} else if v := q.Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

type PaginatedConsolidatedResponse struct {
	Content     []ConsolidatedResponse `json:"content"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"pageSize"`
	TotalPages  int                    `json:"totalPages"`
	TotalItems  int                    `json:"totalItems"`
	HasNext     bool                   `json:"hasNext"`
	HasPrevious bool                   `json:"hasPrevious"`
}

type ConsolidatedHandler struct {
	store  ConsolidationStore
	logger *slog.Logger
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":  message,
		"status": status,
	})
}
