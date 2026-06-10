package database

import (
	"context"
	"time"

	"go-analyze/internal/metrics"
	"go-analyze/pkg/database/entities"
)

type DBService interface {
	Close() error
	Migrate() error
	HealthCheck(ctx context.Context) error
	SaveSnapshot(ctx context.Context, res *metrics.IndicatorResponse, duration time.Duration) error
	SaveConsolidation(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error
	GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error)
	GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error)
	GetWorkloadItemSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error)
	GetConsolidations(ctx context.Context, limit, offset int, startTime, endTime *time.Time) ([]*entities.ConsolidationModel, int, error)
	GetConsolidatedWorkloads(ctx context.Context, consolidationID int) ([]*entities.ConsolidatedWorkloadSnapshotModel, error)
	GetLastWorkloadSnapshots(ctx context.Context, namespace, pod, container string, limit int, startTime, endTime *time.Time) ([]*entities.HistoricalWorkloadSnapshot, error)
}
