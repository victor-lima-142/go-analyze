package consolidator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/victor-lima-142/go-analyze/pkg/database/entities"
)

type mockConsolidationDB struct {
	getScrapesInRangeFunc                  func(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error)
	getIndicatorSnapshotsForScrapesFunc    func(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error)
	getWorkloadItemSnapshotsForScrapesFunc func(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error)
	saveConsolidationFunc                  func(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error
}

func (m *mockConsolidationDB) GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
	if m.getScrapesInRangeFunc != nil {
		return m.getScrapesInRangeFunc(ctx, start, end)
	}
	return nil, nil
}

func (m *mockConsolidationDB) GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error) {
	if m.getIndicatorSnapshotsForScrapesFunc != nil {
		return m.getIndicatorSnapshotsForScrapesFunc(ctx, scrapeIDs)
	}
	return nil, nil
}

func (m *mockConsolidationDB) GetWorkloadItemSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error) {
	if m.getWorkloadItemSnapshotsForScrapesFunc != nil {
		return m.getWorkloadItemSnapshotsForScrapesFunc(ctx, scrapeIDs)
	}
	return nil, nil
}

func (m *mockConsolidationDB) SaveConsolidation(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error {
	if m.saveConsolidationFunc != nil {
		return m.saveConsolidationFunc(ctx, c, workloads)
	}
	return nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestConsolidator_consolidateAndSave(t *testing.T) {
	logger := newDiscardLogger()

	t.Run("Recalculates ratios from totals (no Simpson bias)", func(t *testing.T) {
		scrape1 := entities.NewScrape(50)
		scrape1.SetID(10)
		scrape1.SetScrapedAt(time.Now().Add(-5 * time.Minute))
		scrape2 := entities.NewScrape(40)
		scrape2.SetID(11)
		scrape2.SetScrapedAt(time.Now().Add(-2 * time.Minute))

		// totals: CPU req=2, used=1.4 -> ratio=0.3 ; mem req=2048, used=1024 -> 0.5
		ind1 := entities.NewIndicatorSnapshot(10, 0.2, 0.4, 0.1, 0.8, 5.0, 1.0, 0.8, 1024, 512, 2.0, 4.0, 1000.0, 500.0)
		ind2 := entities.NewIndicatorSnapshot(11, 0.4, 0.6, 0.3, 0.8, 7.0, 1.0, 0.6, 1024, 512, 2.0, 4.0, 1000.0, 500.0)

		limit := 2.0
		limitBytes := 2048.0
		wl1 := entities.NewWorkloadItemSnapshot(10, "default", "pod-abc", "app", 1.0, 0.8, &limit, 1024, 512, &limitBytes, 0.2, 0.5, 5.0)
		wl2 := entities.NewWorkloadItemSnapshot(11, "default", "pod-abc", "app", 1.0, 0.6, &limit, 1024, 256, &limitBytes, 0.4, 0.7, 5.0)

		var savedConsolidation *entities.ConsolidationModel
		var savedWorkloads []*entities.ConsolidatedWorkloadSnapshotModel

		dbMock := &mockConsolidationDB{
			getScrapesInRangeFunc: func(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
				return []*entities.ScrapeModel{scrape1, scrape2}, nil
			},
			getIndicatorSnapshotsForScrapesFunc: func(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error) {
				return []*entities.IndicatorSnapshotModel{ind1, ind2}, nil
			},
			getWorkloadItemSnapshotsForScrapesFunc: func(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error) {
				return []*entities.WorkloadItemSnapshotModel{wl1, wl2}, nil
			},
			saveConsolidationFunc: func(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error {
				savedConsolidation = c
				savedWorkloads = workloads
				return nil
			},
		}

		c := NewConsolidator(dbMock, 1*time.Minute, 10*time.Minute, logger, nil)
		c.consolidateAndSave(context.Background())

		if savedConsolidation == nil {
			t.Fatal("expected SaveConsolidation to be called")
		}

		if !almostEqual(savedConsolidation.CPUWasteRatio(), 0.3) {
			t.Errorf("expected recalculated CPU waste 0.3, got %f", savedConsolidation.CPUWasteRatio())
		}
		if !almostEqual(savedConsolidation.MemWasteRatio(), 0.5) {
			t.Errorf("expected recalculated mem waste 0.5, got %f", savedConsolidation.MemWasteRatio())
		}
		if !almostEqual(savedConsolidation.HPAEfficiency(), 0.5) {
			t.Errorf("expected HPA efficiency 0.5, got %f", savedConsolidation.HPAEfficiency())
		}
		if !almostEqual(savedConsolidation.PVCWasteRatio(), 0.5) {
			t.Errorf("expected PVC waste 0.5, got %f", savedConsolidation.PVCWasteRatio())
		}

		if len(savedWorkloads) != 1 {
			t.Fatalf("expected 1 aggregated workload, got %d", len(savedWorkloads))
		}
		sw := savedWorkloads[0]
		if sw.CPUUsedCores() != 0.7 {
			t.Errorf("expected avg cpu used 0.7, got %f", sw.CPUUsedCores())
		}
		// avg req=1, avg used=0.7 -> ratio = 0.3
		if !almostEqual(sw.CPUWasteRatio(), 0.3) {
			t.Errorf("expected workload cpu waste recalc 0.3, got %f", sw.CPUWasteRatio())
		}
	})

	t.Run("No Scrapes Found Does Not Consolidate", func(t *testing.T) {
		saveCalled := false
		dbMock := &mockConsolidationDB{
			getScrapesInRangeFunc: func(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
				return []*entities.ScrapeModel{}, nil
			},
			saveConsolidationFunc: func(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error {
				saveCalled = true
				return nil
			},
		}
		c := NewConsolidator(dbMock, 1*time.Minute, 10*time.Minute, logger, nil)
		c.consolidateAndSave(context.Background())
		if saveCalled {
			t.Fatal("expected SaveConsolidation to not be called")
		}
	})

	t.Run("DB Scrape Retrieval Error Handles Gracefully", func(t *testing.T) {
		dbMock := &mockConsolidationDB{
			getScrapesInRangeFunc: func(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
				return nil, errors.New("query timed out")
			},
		}
		c := NewConsolidator(dbMock, 1*time.Minute, 10*time.Minute, logger, nil)
		c.consolidateAndSave(context.Background())
	})
}

func TestConsolidator_Start(t *testing.T) {
	logger := newDiscardLogger()
	var saveCount atomic.Int32
	dbMock := &mockConsolidationDB{
		getScrapesInRangeFunc: func(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
			return []*entities.ScrapeModel{}, nil
		},
		saveConsolidationFunc: func(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error {
			saveCount.Add(1)
			return nil
		},
	}
	c := NewConsolidator(dbMock, 10*time.Millisecond, 10*time.Minute, logger, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go c.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
	_ = saveCount.Load()
}
