package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type PromPinger interface {
	QueryInstant(ctx context.Context, query string) (any, error)
}

// HealthHandler validates DB connectivity and Prometheus reachability. It uses
// a short PromQL probe (`up`) to confirm that the metrics backend responds.
type HealthHandler struct {
	db      HealthChecker
	probe   func(ctx context.Context) error
	logger  *slog.Logger
	timeout time.Duration
}

func NewHealthHandler(db HealthChecker, probe func(ctx context.Context) error, logger *slog.Logger) *HealthHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &HealthHandler{db: db, probe: probe, logger: logger, timeout: 3 * time.Second}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	status := map[string]string{"db": "ok", "prometheus": "ok"}
	httpStatus := http.StatusOK

	if err := h.db.HealthCheck(ctx); err != nil {
		status["db"] = "fail: " + err.Error()
		httpStatus = http.StatusServiceUnavailable
	}
	if h.probe != nil {
		if err := h.probe(ctx); err != nil {
			status["prometheus"] = "fail: " + err.Error()
			httpStatus = http.StatusServiceUnavailable
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if httpStatus == http.StatusOK {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "checks": status})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "degraded", "checks": status})
}
