package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/metrics"
	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

func TestWorkloadDetailsHandler_ServeHTTP(t *testing.T) {
	logger := newDiscardLogger()

	t.Run("Missing Parameters", func(t *testing.T) {
		handler := NewWorkloadDetailsHandler(&mockCalculator{}, &mockConsolidationStore{}, "10m", logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workloads/details?namespace=default", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("Success with Live and Historical data", func(t *testing.T) {
		calc := &mockCalculator{
			calculateFunc: func(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error) {
				return &metrics.IndicatorResponse{
					Items: []metrics.WorkloadItem{
						{
							Namespace: "default", Pod: "my-pod-123", Container: "nginx",
							CPURequestedCores: 0.5, CPUUsedCores: 0.25, CPULimitCores: 1.0,
							CPUWasteRatio:        0.5,
							MemoryRequestedBytes: 1024, MemoryUsedBytes: 512, MemoryLimitBytes: 2048,
							MemWasteRatio:            0.5,
							ProjectedMonthlyWasteUSD: 5.0,
							OOMRiskScore:             0.25,
						},
					},
				}, nil
			},
		}

		limit := 1.0
		limitBytes := float64(2048)
		store := &mockConsolidationStore{
			getLastWorkloadSnapshotsFunc: func(ctx context.Context, namespace, pod, container string, limitParam int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error) {
				snap := entities.NewConsolidatedWorkloadSnapshot(1, "default", "my-pod-123", "nginx", 0.5, 0.25, &limit, 1024, 512, &limitBytes, 0.5, 0.5, 5.0)
				snap.SetOOMRiskScore(0.25)
				return []*entities.HistoricalWorkloadSnapshot{
					{ConsolidatedAt: time.Now().Add(-2 * time.Hour), Snapshot: snap},
					{ConsolidatedAt: time.Now().Add(-1 * time.Hour), Snapshot: snap},
				}, nil
			},
		}

		handler := NewWorkloadDetailsHandler(calc, store, "10m", logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workloads/details?namespace=default&container=nginx&pod=my-pod-123", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		var resp WorkloadDetailsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Current == nil || resp.Current.CPUWasteRatio != 0.5 || resp.Current.OOMRiskScore != 0.25 {
			t.Errorf("current wrong: %+v", resp.Current)
		}
		if resp.Historical == nil || resp.Historical.Last10ConsolidationsCount != 2 {
			t.Errorf("historical wrong: %+v", resp.Historical)
		}
	})

	t.Run("Date filtering via start_at/end_at", func(t *testing.T) {
		calc := &mockCalculator{
			calculateFunc: func(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error) {
				return &metrics.IndicatorResponse{}, nil
			},
		}
		var receivedStart, receivedEnd *time.Time
		var receivedLimit int

		store := &mockConsolidationStore{
			getLastWorkloadSnapshotsFunc: func(ctx context.Context, namespace, pod, container string, limitParam int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error) {
				receivedStart = startTime
				receivedEnd = endTime
				receivedLimit = limitParam
				return []*entities.HistoricalWorkloadSnapshot{}, nil
			},
		}

		handler := NewWorkloadDetailsHandler(calc, store, "10m", logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workloads/details?namespace=default&container=nginx&pod=my-pod-123&start_at=2026-05-11T08:00:00Z&end_at=2026-05-11T09:00:00Z", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if receivedStart == nil || receivedEnd == nil {
			t.Fatal("expected dates passed")
		}
		if receivedLimit != 0 {
			t.Errorf("expected unlimited (0), got %d", receivedLimit)
		}
	})
}
