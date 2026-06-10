package prometheus

import (
	"context"
	"fmt"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/victor-lima-142/go-analyze/internal/metrics"
)

type HTTPPrometheusClient struct {
	api     v1.API
	timeout time.Duration
}

func NewHTTPPrometheusClient(address string, timeout time.Duration) (*HTTPPrometheusClient, error) {
	client, err := promapi.NewClient(promapi.Config{Address: address})
	if err != nil {
		return nil, fmt.Errorf("create prometheus client: %w", err)
	}

	return &HTTPPrometheusClient{
		api:     v1.NewAPI(client),
		timeout: timeout,
	}, nil
}

func (c *HTTPPrometheusClient) QueryInstant(ctx context.Context, query string) ([]metrics.Sample, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	val, _, err := c.api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	vector, ok := val.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected query result type: %T", val)
	}

	samples := make([]metrics.Sample, 0, len(vector))
	for _, elem := range vector {
		labels := make(map[string]string, len(elem.Metric))
		for k, v := range elem.Metric {
			labels[string(k)] = string(v)
		}
		samples = append(samples, metrics.Sample{
			Labels: labels,
			Value:  float64(elem.Value),
		})
	}
	return samples, nil
}

func (c *HTTPPrometheusClient) Query(ctx context.Context, promql string, ts time.Time) ([]MetricSeries, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	val, _, err := c.api.Query(ctx, promql, ts)
	if err != nil {
		return nil, err
	}

	return convertToSeries(val)
}

func (c *HTTPPrometheusClient) QueryRange(ctx context.Context, promql string, r QueryRange) ([]MetricSeries, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	val, _, err := c.api.QueryRange(ctx, promql, v1.Range{
		Start: r.Start,
		End:   r.End,
		Step:  r.Step,
	})
	if err != nil {
		return nil, err
	}

	return convertToSeries(val)
}

func convertToSeries(val model.Value) ([]MetricSeries, error) {
	switch v := val.(type) {
	case model.Vector:
		series := make([]MetricSeries, 0, len(v))
		for _, elem := range v {
			labels := make(map[string]string, len(elem.Metric))
			for lK, lV := range elem.Metric {
				labels[string(lK)] = string(lV)
			}
			series = append(series, MetricSeries{
				MetricName: string(elem.Metric[model.MetricNameLabel]),
				Labels:     labels,
				Points: []MetricPoint{
					{
						Timestamp: elem.Timestamp.Time(),
						Value:     float64(elem.Value),
					},
				},
			})
		}
		return series, nil

	case model.Matrix:
		series := make([]MetricSeries, 0, len(v))
		for _, elem := range v {
			labels := make(map[string]string, len(elem.Metric))
			for lK, lV := range elem.Metric {
				labels[string(lK)] = string(lV)
			}
			points := make([]MetricPoint, 0, len(elem.Values))
			for _, p := range elem.Values {
				points = append(points, MetricPoint{
					Timestamp: p.Timestamp.Time(),
					Value:     float64(p.Value),
				})
			}
			series = append(series, MetricSeries{
				MetricName: string(elem.Metric[model.MetricNameLabel]),
				Labels:     labels,
				Points:     points,
			})
		}
		return series, nil

	default:
		return nil, fmt.Errorf("unsupported model value type: %T", val)
	}
}
