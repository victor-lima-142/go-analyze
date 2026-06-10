package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/observability"
	"github.com/victor-lima-142/go-analyze/pkg/models"
)

type QueryParams = models.QueryParams
type Sample = models.MetricSample
type IndicatorResponse = models.IndicatorResult
type WorkloadItem = models.WorkloadItem
type Indicators = models.Indicators
type Inputs = models.Inputs

type PrometheusClient interface {
	QueryInstant(ctx context.Context, query string) ([]models.MetricSample, error)
}

type CalculatorOptions struct {
	Client             PrometheusClient
	CPUHourlyUSD       float64
	MemoryGiBHourlyUSD float64
	MonthlyHours       float64
	HPAWindow          time.Duration
	CostModelLabel     string
	Logger             *slog.Logger
	Filter             *observability.Filter
}
