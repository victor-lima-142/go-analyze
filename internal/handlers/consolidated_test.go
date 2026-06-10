package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

type mockConsolidationStore struct {
	getConsolidationsFunc        func(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error)
	getConsolidatedWorkloadsFunc func(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error)
	getLastWorkloadSnapshotsFunc func(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error)
}

func (m *mockConsolidationStore) GetConsolidations(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
	if m.getConsolidationsFunc != nil {
		return m.getConsolidationsFunc(ctx, limit, offset, startTime, endTime)
	}
	return nil, 0, nil
}

func (m *mockConsolidationStore) GetConsolidatedWorkloads(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error) {
	if m.getConsolidatedWorkloadsFunc != nil {
		return m.getConsolidatedWorkloadsFunc(ctx, consolidationID)
	}
	return nil, nil
}

func (m *mockConsolidationStore) GetLastWorkloadSnapshots(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error) {
	if m.getLastWorkloadSnapshotsFunc != nil {
		return m.getLastWorkloadSnapshotsFunc(ctx, namespace, pod, container, limit, startTime, endTime)
	}
	return nil, nil
}

func TestConsolidatedHandler_ServeHTTP(t *testing.T) {
	logger := newDiscardLogger()

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		handler := NewConsolidatedHandler(&mockConsolidationStore{}, logger)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/consolidated", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("Database Error Handling", func(t *testing.T) {
		store := &mockConsolidationStore{
			getConsolidationsFunc: func(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
				return nil, 0, errors.New("db disconnect")
			},
		}
		handler := NewConsolidatedHandler(store, logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/consolidated", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("Success with consolidation data", func(t *testing.T) {
		startTime := time.Now().Add(-10 * time.Minute)
		endTime := time.Now()

		cModel := entities.NewConsolidation(
			startTime, endTime, 20,
			0.1, 0.2, 0.3, 0.4, 5.5,
			1.0, 0.8, 1024, 512,
		)
		cModel.SetID(42)
		cModel.SetConsolidatedAt(time.Now())
		cModel.SetOOMRiskScore(0.05)

		var limitParam int
		var offsetParam int = -1

		store := &mockConsolidationStore{
			getConsolidationsFunc: func(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
				limitParam = limit
				offsetParam = offset
				return []*entities.ConsolidationModel{cModel}, 1, nil
			},
			getConsolidatedWorkloadsFunc: func(ctx context.Context, id int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error) {
				if id != 42 {
					t.Errorf("expected 42, got %d", id)
				}
				limitVal := 2.0
				limitBytes := 2048.0
				wModel := entities.NewConsolidatedWorkloadSnapshot(
					42, "heavy", "my-pod", "my-container",
					1.0, 0.5, &limitVal,
					1024.0, 512.0, &limitBytes,
					0.5, 0.5, 2.5,
				)
				wModel.SetID(101)
				wModel.SetOOMRiskScore(0.25)
				return []*entities.ConsolidatedWorkloadSnapshotModel{wModel}, nil
			},
		}
		handler := NewConsolidatedHandler(store, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/consolidated?limit=10", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if limitParam != 10 || offsetParam != 0 {
			t.Errorf("pagination args wrong: limit=%d offset=%d", limitParam, offsetParam)
		}

		var resp PaginatedConsolidatedResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Page != 1 || resp.PageSize != 10 || resp.TotalItems != 1 {
			t.Errorf("pagination header wrong: %+v", resp)
		}
		if len(resp.Content) != 1 || resp.Content[0].ID != 42 {
			t.Fatalf("content wrong: %+v", resp.Content)
		}
		if resp.Content[0].Indicators.OOMRiskScore != 0.05 {
			t.Errorf("OOMRiskScore not propagated: %f", resp.Content[0].Indicators.OOMRiskScore)
		}
		if len(resp.Content[0].Items) != 1 || resp.Content[0].Items[0].OOMRiskScore != 0.25 {
			t.Errorf("workload OOM not propagated: %+v", resp.Content[0].Items)
		}
	})

	t.Run("Period filtering", func(t *testing.T) {
		var receivedStart, receivedEnd *time.Time
		store := &mockConsolidationStore{
			getConsolidationsFunc: func(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error) {
				receivedStart = startTime
				receivedEnd = endTime
				return nil, 0, nil
			},
		}
		handler := NewConsolidatedHandler(store, logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/consolidated?period=1hr", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if receivedStart == nil || receivedEnd == nil {
			t.Fatal("expected window")
		}
		diff := receivedEnd.Sub(*receivedStart)
		if diff < 59*time.Minute || diff > 61*time.Minute {
			t.Errorf("expected ~1h, got %s", diff)
		}
	})
}
