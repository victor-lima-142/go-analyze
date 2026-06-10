package scraper

import (
	"context"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/metrics"
)

type ScraperService interface {
	Start(ctx context.Context)
}

type SnapshotSaver interface {
	SaveSnapshot(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error
}
