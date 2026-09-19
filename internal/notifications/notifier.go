package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/telemetry"
)

type Notification struct {
	Indicator          string        `json:"indicator"`
	Scope              string        `json:"scope"`
	Namespace          string        `json:"namespace,omitempty"`
	Pod                string        `json:"pod,omitempty"`
	Container          string        `json:"container,omitempty"`
	Value              float64       `json:"value"`
	Threshold          float64       `json:"threshold"`
	DurationSustained  time.Duration `json:"duration_sustained"`
	ConfiguredDuration time.Duration `json:"configured_duration"`
	FiredAt            time.Time     `json:"fired_at"`
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
	scope := "workload"
	if n.Namespace == "" && n.Pod == "" && n.Container == "" {
		scope = "global"
	}
	n.Scope = scope
	telemetry.NotificationsTotal.WithLabelValues(n.Indicator, scope, n.Namespace, n.Pod, n.Container).Inc()
	telemetry.NotificationLastTimestamp.WithLabelValues(n.Indicator, scope, n.Namespace, n.Pod, n.Container).Set(float64(n.FiredAt.Unix()))
	telemetry.NotificationSustainedDuration.WithLabelValues(n.Indicator, scope, n.Namespace, n.Pod, n.Container).Set(n.DurationSustained.Seconds())
	s.Logger.Warn("overprovisioning detected",
		"indicator", n.Indicator,
		"scope", scope,
		"namespace", n.Namespace,
		"pod", n.Pod,
		"container", n.Container,
		"value", n.Value,
		"threshold", n.Threshold,
		"sustained_for", n.DurationSustained.String(),
		"signal_latency", (n.DurationSustained - n.ConfiguredDuration).String(),
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
