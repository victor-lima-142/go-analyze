package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type resetDBStub struct{ calls int }

func (s *resetDBStub) TruncateData(context.Context) error { s.calls++; return nil }

type resetTrackerStub struct{ calls int }

func (s *resetTrackerStub) Reset() { s.calls++ }

func TestExperimentResetAuthorization(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		header  string
		want    int
		calls   int
	}{
		{"disabled", false, "Bearer secret", http.StatusNotFound, 0},
		{"missing token", true, "", http.StatusUnauthorized, 0},
		{"invalid token", true, "Bearer wrong", http.StatusUnauthorized, 0},
		{"valid token", true, "Bearer secret", http.StatusOK, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, tracker := &resetDBStub{}, &resetTrackerStub{}
			h := NewExperimentResetHandler(db, tracker, tc.enabled, "secret", newDiscardLogger())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/experiment/reset", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want || db.calls != tc.calls || tracker.calls != tc.calls {
				t.Fatalf("code=%d db=%d tracker=%d", rec.Code, db.calls, tracker.calls)
			}
		})
	}
}
