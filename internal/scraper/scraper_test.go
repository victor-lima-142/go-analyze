package scraper

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"go-analyze/internal/metrics"
	"go-analyze/internal/observability"
	"go-analyze/pkg/models"
)

type mockPromClient struct {
	queryInstantFunc func(ctx context.Context, query string) ([]models.MetricSample, error)
}

func (m *mockPromClient) QueryInstant(ctx context.Context, query string) ([]models.MetricSample, error) {
	if m.queryInstantFunc != nil {
		return m.queryInstantFunc(ctx, query)
	}
	return nil, nil
}

type mockSnapshotSaver struct {
	saveSnapshotFunc func(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error
}

func (m *mockSnapshotSaver) SaveSnapshot(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error {
	if m.saveSnapshotFunc != nil {
		return m.saveSnapshotFunc(ctx, res, duration)
	}
	return nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestCalculator(promMock metrics.PrometheusClient) *metrics.Calculator {
	return metrics.NewCalculator(metrics.CalculatorOptions{
		Client:             promMock,
		CPUHourlyUSD:       0.05,
		MemoryGiBHourlyUSD: 0.01,
		MonthlyHours:       730,
		Logger:             newTestLogger(),
		Filter:             observability.DefaultFilter(),
	})
}

func TestScraper_scrapeAndSave(t *testing.T) {
	logger := newTestLogger()

	t.Run("Success Scrape and Persist", func(t *testing.T) {
		promMock := &mockPromClient{
			queryInstantFunc: func(ctx context.Context, query string) ([]models.MetricSample, error) {
				return []models.MetricSample{
					{
						Labels: map[string]string{"namespace": "default", "pod": "my-pod", "container": "my-container"},
						Value:  1.0,
					},
				}, nil
			},
		}
		calc := newTestCalculator(promMock)

		var savedRes *metrics.IndicatorResponse
		var savedDuration time.Duration
		dbMock := &mockSnapshotSaver{
			saveSnapshotFunc: func(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error {
				savedRes = res
				savedDuration = duration
				return nil
			},
		}

		scraper := NewScraper(calc, dbMock, 1*time.Second, "10m", logger)
		scraper.scrapeAndSave(context.Background())

		if savedRes == nil {
			t.Fatal("expected SaveSnapshot to be called")
		}
		if len(savedRes.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(savedRes.Items))
		}
		if savedDuration <= 0 {
			t.Errorf("expected positive duration, got %s", savedDuration)
		}
	})

	t.Run("Calculation Error Handles Gracefully", func(t *testing.T) {
		promMock := &mockPromClient{
			queryInstantFunc: func(ctx context.Context, query string) ([]models.MetricSample, error) {
				return nil, errors.New("prometheus timeout")
			},
		}
		calc := newTestCalculator(promMock)

		dbCalled := false
		dbMock := &mockSnapshotSaver{
			saveSnapshotFunc: func(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error {
				dbCalled = true
				return nil
			},
		}
		scraper := NewScraper(calc, dbMock, 1*time.Second, "10m", logger)
		scraper.scrapeAndSave(context.Background())

		if dbCalled {
			t.Fatal("SaveSnapshot should not run on calc error")
		}
	})
}

func TestScraper_Start(t *testing.T) {
	logger := newTestLogger()

	promMock := &mockPromClient{
		queryInstantFunc: func(ctx context.Context, query string) ([]models.MetricSample, error) {
			return []models.MetricSample{}, nil
		},
	}
	calc := newTestCalculator(promMock)

	var saveCount atomic.Int32
	dbMock := &mockSnapshotSaver{
		saveSnapshotFunc: func(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error {
			saveCount.Add(1)
			return nil
		},
	}

	scraper := NewScraper(calc, dbMock, 10*time.Millisecond, "10m", logger)
	ctx, cancel := context.WithCancel(context.Background())

	go scraper.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if saveCount.Load() < 2 {
		t.Errorf("expected at least 2 saves, got %d", saveCount.Load())
	}
}
