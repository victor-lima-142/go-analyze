package prometheus

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type mockAPI struct {
	v1.API
	queryFunc      func(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error)
	queryRangeFunc func(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error)
}

func (m *mockAPI) Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, ts, opts...)
	}
	return nil, nil, nil
}

func (m *mockAPI) QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	if m.queryRangeFunc != nil {
		return m.queryRangeFunc(ctx, query, r, opts...)
	}
	return nil, nil, nil
}

func TestHTTPPrometheusClient_QueryInstant(t *testing.T) {
	t.Run("Success Vector Result", func(t *testing.T) {
		mAPI := &mockAPI{
			queryFunc: func(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
				v := model.Vector{
					&model.Sample{
						Metric: model.Metric{
							"namespace": "test-ns",
							"pod":       "my-pod",
						},
						Value:     3.14,
						Timestamp: model.TimeFromUnix(1715000000),
					},
				}
				return v, nil, nil
			},
		}

		client := &HTTPPrometheusClient{
			api:     mAPI,
			timeout: 5 * time.Second,
		}

		samples, err := client.QueryInstant(context.Background(), "up")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(samples) != 1 {
			t.Fatalf("expected 1 sample, got %d", len(samples))
		}

		if samples[0].Value != 3.14 {
			t.Errorf("expected sample value 3.14, got %f", samples[0].Value)
		}

		if samples[0].Labels["namespace"] != "test-ns" {
			t.Errorf("expected label namespace test-ns, got %s", samples[0].Labels["namespace"])
		}
	})

	t.Run("Query API Error", func(t *testing.T) {
		mAPI := &mockAPI{
			queryFunc: func(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
				return nil, nil, errors.New("prometheus connection lost")
			},
		}

		client := &HTTPPrometheusClient{
			api:     mAPI,
			timeout: 5 * time.Second,
		}

		_, err := client.QueryInstant(context.Background(), "up")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Unexpected Result Type", func(t *testing.T) {
		mAPI := &mockAPI{
			queryFunc: func(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
				// Returning Matrix instead of Vector for Instant query
				m := model.Matrix{}
				return m, nil, nil
			},
		}

		client := &HTTPPrometheusClient{
			api:     mAPI,
			timeout: 5 * time.Second,
		}

		_, err := client.QueryInstant(context.Background(), "up")
		if err == nil {
			t.Fatal("expected error due to unexpected type, got nil")
		}
	})
}

func TestHTTPPrometheusClient_QueryAndQueryRange(t *testing.T) {
	t.Run("Query Success with Matrix", func(t *testing.T) {
		mAPI := &mockAPI{
			queryFunc: func(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
				m := model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "container_memory_usage_bytes",
							"container":            "my-container",
						},
						Values: []model.SamplePair{
							{
								Timestamp: model.TimeFromUnix(1715000000),
								Value:     1024,
							},
						},
					},
				}
				return m, nil, nil
			},
		}

		client := &HTTPPrometheusClient{
			api:     mAPI,
			timeout: 5 * time.Second,
		}

		seriesList, err := client.Query(context.Background(), "container_memory_usage_bytes", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seriesList) != 1 {
			t.Fatalf("expected 1 series, got %d", len(seriesList))
		}

		if seriesList[0].MetricName != "container_memory_usage_bytes" {
			t.Errorf("expected metric name to be container_memory_usage_bytes, got %s", seriesList[0].MetricName)
		}

		if len(seriesList[0].Points) != 1 || seriesList[0].Points[0].Value != 1024 {
			t.Errorf("expected 1 point with value 1024, got %v", seriesList[0].Points)
		}
	})

	t.Run("QueryRange Success", func(t *testing.T) {
		mAPI := &mockAPI{
			queryRangeFunc: func(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
				m := model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "cpu_usage",
						},
						Values: []model.SamplePair{
							{
								Timestamp: model.TimeFromUnix(1715000000),
								Value:     0.75,
							},
						},
					},
				}
				return m, nil, nil
			},
		}

		client := &HTTPPrometheusClient{
			api:     mAPI,
			timeout: 5 * time.Second,
		}

		qr := QueryRange{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now(),
			Step:  1 * time.Minute,
		}

		seriesList, err := client.QueryRange(context.Background(), "cpu_usage", qr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seriesList) != 1 {
			t.Fatalf("expected 1 series, got %d", len(seriesList))
		}

		if seriesList[0].Points[0].Value != 0.75 {
			t.Errorf("expected cpu_usage to be 0.75, got %f", seriesList[0].Points[0].Value)
		}
	})
}
