package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/telemetry"
)

type Notification struct {
	Indicator         string        `json:"indicator"`
	Namespace         string        `json:"namespace,omitempty"`
	Pod               string        `json:"pod,omitempty"`
	Container         string        `json:"container,omitempty"`
	Value             float64       `json:"value"`
	Threshold         float64       `json:"threshold"`
	DurationSustained time.Duration `json:"duration_sustained"`
	FiredAt           time.Time     `json:"fired_at"`
}

type Notifier interface {
	Notify(ctx context.Context, n Notification)
}

type SlogNotifier struct {
	Logger *slog.Logger
}

func NewSlogNotifier(logger *slog.Logger) *SlogNotifier {
	return &SlogNotifier{Logger: logger}
}

func (s *SlogNotifier) Notify(_ context.Context, n Notification) {
	telemetry.NotificationsTotal.WithLabelValues(n.Indicator).Inc()
	s.Logger.Warn("overprovisioning detected",
		"indicator", n.Indicator,
		"namespace", n.Namespace,
		"pod", n.Pod,
		"container", n.Container,
		"value", n.Value,
		"threshold", n.Threshold,
		"sustained_for", n.DurationSustained.String(),
	)
}

type MultiNotifier struct {
	Sinks []Notifier
}

func (m *MultiNotifier) Notify(ctx context.Context, n Notification) {
	for _, s := range m.Sinks {
		s.Notify(ctx, n)
	}
}
