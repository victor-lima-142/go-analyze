package notifications

import (
	"context"
	"sync"
	"time"
)

type IndicatorKey struct {
	Indicator string
	Namespace string
	Pod       string
	Container string
}

type IndicatorRule struct {
	Threshold        float64
	RequiredDuration time.Duration
	Comparator       func(value, threshold float64) bool
}

type trackedState struct {
	above          bool
	firstAbove     time.Time
	lastValue      float64
	lastNotifiedAt time.Time
	lastObservedAt time.Time
}

type Tracker struct {
	rules    map[string]IndicatorRule
	mu       sync.Mutex
	states   map[IndicatorKey]*trackedState
	notifier Notifier
	now      func() time.Time
	cooldown time.Duration
	maxGap   time.Duration
}

func NewTracker(rules map[string]IndicatorRule, notifier Notifier, cooldown time.Duration, maxGap ...time.Duration) *Tracker {
	var gap time.Duration
	if len(maxGap) > 0 {
		gap = maxGap[0]
	}
	return &Tracker{
		rules:    rules,
		states:   make(map[IndicatorKey]*trackedState),
		notifier: notifier,
		now:      time.Now,
		cooldown: cooldown,
		maxGap:   gap,
	}
}

// Observe updates the tracker state and emits a notification when the value has
// been sustained above the threshold for at least the rule's RequiredDuration.
func (t *Tracker) Observe(ctx context.Context, key IndicatorKey, value float64) {
	rule, ok := t.rules[key.Indicator]
	if !ok {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	state, exists := t.states[key]
	if !exists {
		state = &trackedState{}
		t.states[key] = state
	}
	state.lastValue = value
	if state.above && t.maxGap > 0 && !state.lastObservedAt.IsZero() && now.Sub(state.lastObservedAt) > t.maxGap {
		state.above = false
		state.firstAbove = time.Time{}
	}
	state.lastObservedAt = now

	exceeds := rule.Comparator(value, rule.Threshold)
	if !exceeds {
		state.above = false
		state.firstAbove = time.Time{}
		return
	}

	if !state.above {
		state.above = true
		state.firstAbove = now
		return
	}

	sustained := now.Sub(state.firstAbove)
	if sustained < rule.RequiredDuration {
		return
	}
	if !state.lastNotifiedAt.IsZero() && now.Sub(state.lastNotifiedAt) < t.cooldown {
		return
	}
	state.lastNotifiedAt = now
	t.notifier.Notify(ctx, Notification{
		Indicator:          key.Indicator,
		Namespace:          key.Namespace,
		Pod:                key.Pod,
		Container:          key.Container,
		Value:              value,
		Threshold:          rule.Threshold,
		DurationSustained:  sustained,
		ConfiguredDuration: rule.RequiredDuration,
		FiredAt:            now,
	})
}

// SnapshotState exposes the current sustained state for inspection/testing.
func (t *Tracker) SnapshotState(key IndicatorKey) (above bool, firstAbove time.Time, lastValue float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.states[key]
	if !ok {
		return false, time.Time{}, 0
	}
	return s.above, s.firstAbove, s.lastValue
}

// Reset clears all tracked states, effectively restarting the duration
// count for all indicators. This is used between experimental runs.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.states = make(map[IndicatorKey]*trackedState)
}

func GreaterThan(value, threshold float64) bool { return value > threshold }
func LessThan(value, threshold float64) bool    { return value < threshold }
