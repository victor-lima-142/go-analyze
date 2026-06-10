package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// ResetDB defines the persistence operations needed to clear experimental state.
type ResetDB interface {
	TruncateData(ctx context.Context) error
}

// ResetTracker defines the in-memory operations needed to clear experimental state.
type ResetTracker interface {
	Reset()
}

// ExperimentResetHandler clears both the database and the notification trackers
// to allow clean, repeatable experimental runs.
type ExperimentResetHandler struct {
	db      ResetDB
	tracker ResetTracker
	logger  *slog.Logger
}

func NewExperimentResetHandler(db ResetDB, tracker ResetTracker, logger *slog.Logger) *ExperimentResetHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExperimentResetHandler{
		db:      db,
		tracker: tracker,
		logger:  logger,
	}
}

// ServeHTTP handles POST /api/v1/experiment/reset
// @Summary      Resetar dados do experimento
// @Description  Limpa as tabelas de histórico do banco de dados e zera o estado interno dos trackers de notificação para uma nova rodada de teste.
// @Tags         experiment
// @Accept       json
// @Produce      json
// @Success      200      {object}  map[string]string
// @Failure      405      {object}  map[string]any
// @Failure      500      {object}  map[string]any
// @Router       /api/v1/experiment/reset [post]
func (h *ExperimentResetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.logger.Info("experiment reset requested: truncating data and resetting trackers")

	if err := h.db.TruncateData(r.Context()); err != nil {
		h.logger.Error("failed to truncate database data", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to truncate database")
		return
	}

	h.tracker.Reset()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "experiment data truncated and trackers reset",
	})
}
