package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-analyze/internal/metrics"
)

type mockCalculator struct {
	calculateFunc func(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error)
}

func (m *mockCalculator) Calculate(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error) {
	if m.calculateFunc != nil {
		return m.calculateFunc(ctx, p)
	}
	return nil, nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestIndicatorHandler_ServeHTTP_Pagination(t *testing.T) {
	logger := newDiscardLogger()

	t.Run("Success with paginated items", func(t *testing.T) {
		calc := &mockCalculator{
			calculateFunc: func(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error) {
				return &metrics.IndicatorResponse{
					Filters:    p,
					Indicators: metrics.Indicators{CPUWasteRatio: 0.5},
					Inputs:     metrics.Inputs{CPURequestedCores: 2.0},
					Items: []metrics.WorkloadItem{
						{Namespace: "ns1", Pod: "pod1", Container: "c1"},
						{Namespace: "ns1", Pod: "pod2", Container: "c2"},
						{Namespace: "ns1", Pod: "pod3", Container: "c3"},
					},
				}, nil
			},
		}
		handler := NewIndicatorHandler(calc, "5m", logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators?page=1&pageSize=2", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}

		var resp PaginatedIndicatorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Page != 1 || resp.PageSize != 2 || resp.TotalItems != 3 || resp.TotalPages != 2 {
			t.Errorf("pagination header mismatch: %+v", resp)
		}
		if !resp.HasNext || resp.HasPrevious {
			t.Errorf("expected hasNext=true hasPrevious=false, got %+v", resp)
		}
		if len(resp.Content) != 2 || resp.Content[0].Pod != "pod1" {
			t.Errorf("page 1 content wrong: %+v", resp.Content)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/indicators?page=2&pageSize=2", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Page != 2 || resp.HasNext || !resp.HasPrevious || len(resp.Content) != 1 {
			t.Errorf("page 2 wrong: %+v", resp)
		}
	})

	t.Run("pageSize clamped to maxPageSize", func(t *testing.T) {
		calc := &mockCalculator{
			calculateFunc: func(ctx context.Context, p metrics.QueryParams) (*metrics.IndicatorResponse, error) {
				return &metrics.IndicatorResponse{}, nil
			},
		}
		handler := NewIndicatorHandler(calc, "5m", logger)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators?pageSize=999999", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var resp PaginatedIndicatorResponse
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp.PageSize != maxPageSize {
			t.Errorf("expected pageSize clamped to %d, got %d", maxPageSize, resp.PageSize)
		}
	})
}
