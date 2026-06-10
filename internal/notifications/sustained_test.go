package notifications

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

type captureNotifier struct {
	got []Notification
}

func (c *captureNotifier) Notify(_ context.Context, n Notification) {
	c.got = append(c.got, n)
}

func newTrackerWithClock(rules map[string]IndicatorRule, notifier Notifier, cooldown time.Duration, clock func() time.Time) *Tracker {
	t := NewTracker(rules, notifier, cooldown)
	t.now = clock
	return t
}

func TestTracker_SustainedDetectionFires(t *testing.T) {
	notifier := &captureNotifier{}
	clock := time.Now()
	tracker := newTrackerWithClock(map[string]IndicatorRule{
		"cpu_waste_ratio": {Threshold: 0.5, RequiredDuration: 3 * time.Minute, Comparator: GreaterThan},
	}, notifier, time.Hour, func() time.Time { return clock })

	key := IndicatorKey{Indicator: "cpu_waste_ratio", Namespace: "heavy", Pod: "p", Container: "c"}

	tracker.Observe(context.Background(), key, 0.6) // first above
	if len(notifier.got) != 0 {
		t.Fatalf("expected no notification yet, got %d", len(notifier.got))
	}

	clock = clock.Add(2 * time.Minute)
	tracker.Observe(context.Background(), key, 0.7) // still below required duration
	if len(notifier.got) != 0 {
		t.Fatalf("expected no notification yet, got %d", len(notifier.got))
	}

	clock = clock.Add(2 * time.Minute) // total 4 minutes above
	tracker.Observe(context.Background(), key, 0.8)
	if len(notifier.got) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.got))
	}
	if notifier.got[0].Indicator != "cpu_waste_ratio" {
		t.Errorf("indicator mismatch: %s", notifier.got[0].Indicator)
	}
}

func TestTracker_BelowThresholdResets(t *testing.T) {
	notifier := &captureNotifier{}
	clock := time.Now()
	tracker := newTrackerWithClock(map[string]IndicatorRule{
		"cpu_waste_ratio": {Threshold: 0.5, RequiredDuration: 1 * time.Minute, Comparator: GreaterThan},
	}, notifier, time.Hour, func() time.Time { return clock })

	key := IndicatorKey{Indicator: "cpu_waste_ratio"}
	tracker.Observe(context.Background(), key, 0.6)
	clock = clock.Add(30 * time.Second)
	tracker.Observe(context.Background(), key, 0.3) // dips below
	clock = clock.Add(2 * time.Minute)
	tracker.Observe(context.Background(), key, 0.55) // re-enters above, restart clock
	if len(notifier.got) != 0 {
		t.Fatalf("expected reset, got %d notifications", len(notifier.got))
	}
}

func TestTracker_CooldownPreventsSpam(t *testing.T) {
	notifier := &captureNotifier{}
	clock := time.Now()
	tracker := newTrackerWithClock(map[string]IndicatorRule{
		"cpu_waste_ratio": {Threshold: 0.5, RequiredDuration: 1 * time.Minute, Comparator: GreaterThan},
	}, notifier, 10*time.Minute, func() time.Time { return clock })

	key := IndicatorKey{Indicator: "cpu_waste_ratio"}
	tracker.Observe(context.Background(), key, 0.7)
	clock = clock.Add(2 * time.Minute)
	tracker.Observe(context.Background(), key, 0.7) // fires
	tracker.Observe(context.Background(), key, 0.7) // suppressed
	clock = clock.Add(1 * time.Minute)
	tracker.Observe(context.Background(), key, 0.7) // still in cooldown
	if len(notifier.got) != 1 {
		t.Fatalf("expected 1 notification due to cooldown, got %d", len(notifier.got))
	}

	clock = clock.Add(11 * time.Minute)
	tracker.Observe(context.Background(), key, 0.7) // cooldown expired
	if len(notifier.got) != 2 {
		t.Fatalf("expected 2 notifications after cooldown, got %d", len(notifier.got))
	}
}

func TestSlogNotifier_DoesNotPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := NewSlogNotifier(logger)
	n.Notify(context.Background(), Notification{Indicator: "x", Value: 1, Threshold: 0.5, FiredAt: time.Now()})
}
