package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHealthDB struct {
	err error
}

func (f *fakeHealthDB) HealthCheck(ctx context.Context) error { return f.err }

func TestHealthHandler_AllOk(t *testing.T) {
	handler := NewHealthHandler(&fakeHealthDB{}, func(ctx context.Context) error { return nil }, newDiscardLogger())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthHandler_DBFailure(t *testing.T) {
	handler := NewHealthHandler(&fakeHealthDB{err: errors.New("conn refused")}, func(ctx context.Context) error { return nil }, newDiscardLogger())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestHealthHandler_PrometheusFailure(t *testing.T) {
	handler := NewHealthHandler(&fakeHealthDB{}, func(ctx context.Context) error { return errors.New("timeout") }, newDiscardLogger())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
