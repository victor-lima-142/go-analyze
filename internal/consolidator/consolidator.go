package consolidator

import (
	"context"
	"log/slog"
	"time"

	"go-analyze/internal/notifications"
	"go-analyze/internal/numeric"
	"go-analyze/internal/telemetry"
	"go-analyze/pkg/database/entities"
)

type ConsolidationDB interface {
	GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error)
	GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error)
	GetWorkloadItemSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.WorkloadItemSnapshotModel, error)
	SaveConsolidation(ctx context.Context, c *entities.ConsolidationModel, workloads []*entities.ConsolidatedWorkloadSnapshotModel) error
}

type Consolidator struct {
	db       ConsolidationDB
	interval time.Duration
	window   time.Duration
	logger   *slog.Logger
	tracker  *notifications.Tracker
}

func NewConsolidator(db ConsolidationDB, interval, window time.Duration, logger *slog.Logger, tracker *notifications.Tracker) *Consolidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consolidator{
		db:       db,
		interval: interval,
		window:   window,
		logger:   logger,
		tracker:  tracker,
	}
}

func (c *Consolidator) Start(ctx context.Context) {
	c.logger.Info("starting background consolidator", "interval", c.interval, "window", c.window)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.consolidateAndSave(ctx)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("stopping background consolidator: context cancelled")
			return
		case <-ticker.C:
			c.consolidateAndSave(ctx)
		}
	}
}

func ratioOver(numeratorPositive bool, num, den float64) float64 {
	if den <= 0 {
		return 0
	}
	value := num / den
	if numeratorPositive && value < 0 {
		value = 0
	}
	return numeric.Round4(value)
}

func (c *Consolidator) consolidateAndSave(ctx context.Context) {
	telemetry.ConsolidationsTotal.Inc()
	tracer := telemetry.Tracer("consolidator")
	ctx, span := tracer.Start(ctx, "consolidate_cycle")
	defer span.End()

	c.logger.Debug("consolidation cycle started")
	now := time.Now()
	start := now.Add(-c.window)

	scrapes, err := c.db.GetScrapesInRange(ctx, start, now)
	if err != nil {
		telemetry.ConsolidationErrors.Inc()
		c.logger.Error("failed to retrieve scrapes in range", "error", err)
		return
	}
	scrapesCount := len(scrapes)
	if scrapesCount == 0 {
		c.logger.Debug("no scrapes found in window to consolidate")
		return
	}

	scrapeIDs := make([]int, scrapesCount)
	for i, s := range scrapes {
		scrapeIDs[i] = s.ID()
	}

	indicators, err := c.db.GetIndicatorSnapshotsForScrapes(ctx, scrapeIDs)
	if err != nil {
		telemetry.ConsolidationErrors.Inc()
		c.logger.Error("failed to retrieve indicator snapshots", "error", err)
		return
	}
	if len(indicators) == 0 {
		c.logger.Debug("no indicator snapshots found")
		return
	}

	workloads, err := c.db.GetWorkloadItemSnapshotsForScrapes(ctx, scrapeIDs)
	if err != nil {
		telemetry.ConsolidationErrors.Inc()
		c.logger.Error("failed to retrieve workload snapshots", "error", err)
		return
	}

	var sumCPUReq, sumCPUUsed, sumMemReq, sumMemUsed float64
	var sumPVCCap, sumPVCUsed float64
	var sumHPAAvg, sumHPAMax float64
	var sumProjected, sumMemLimit float64
	for _, ind := range indicators {
		sumCPUReq += ind.TotalCPURequested()
		sumCPUUsed += ind.TotalCPUUsed()
		sumMemReq += ind.TotalMemoryRequested()
		sumMemUsed += ind.TotalMemoryUsed()
		sumPVCCap += ind.PVCCapacityBytes()
		sumPVCUsed += ind.PVCUsedBytes()
		sumHPAAvg += ind.HPAAvgReplicas()
		sumHPAMax += ind.HPAMaxReplicas()
		sumProjected += ind.ProjectedMonthlyWasteUSD()
	}
	indCount := float64(len(indicators))
	avgProjected := numeric.Round4(sumProjected / indCount)

	avgCPUReq := sumCPUReq / indCount
	avgCPUUsed := sumCPUUsed / indCount
	avgMemReq := sumMemReq / indCount
	avgMemUsed := sumMemUsed / indCount

	cpuWaste := ratioOver(true, sumCPUReq-sumCPUUsed, sumCPUReq)
	memWaste := ratioOver(true, sumMemReq-sumMemUsed, sumMemReq)
	pvcWaste := ratioOver(true, sumPVCCap-sumPVCUsed, sumPVCCap)
	hpaEff := ratioOver(false, sumHPAAvg, sumHPAMax)
	oomRisk := ratioOver(false, sumMemUsed, sumMemLimit)

	consolidation := entities.NewConsolidation(
		start, now, scrapesCount,
		cpuWaste, memWaste, pvcWaste, hpaEff, avgProjected,
		avgCPUReq, avgCPUUsed, avgMemReq, avgMemUsed,
	)
	consolidation.SetOOMRiskScore(oomRisk)

	type wlKey struct{ namespace, pod, container string }
	type wlStats struct {
		cpuReqSum     float64
		cpuUsedSum    float64
		cpuLimitSum   float64
		cpuLimitCount int
		memReqSum     float64
		memUsedSum    float64
		memLimitSum   float64
		memLimitCount int
		projectedSum  float64
		count         int
	}

	grouped := make(map[wlKey]*wlStats)
	for _, wl := range workloads {
		key := wlKey{namespace: wl.Namespace(), pod: wl.Pod(), container: wl.Container()}
		stats, ok := grouped[key]
		if !ok {
			stats = &wlStats{}
			grouped[key] = stats
		}

		stats.count++
		stats.cpuReqSum += wl.CPURequestedCores()
		stats.cpuUsedSum += wl.CPUUsedCores()
		if wl.CPULimitCores() != nil {
			stats.cpuLimitSum += *wl.CPULimitCores()
			stats.cpuLimitCount++
		}
		stats.memReqSum += wl.MemoryRequestedBytes()
		stats.memUsedSum += wl.MemoryUsedBytes()
		if wl.MemoryLimitBytes() != nil {
			stats.memLimitSum += *wl.MemoryLimitBytes()
			stats.memLimitCount++
		}
		stats.projectedSum += wl.ProjectedMonthlyWasteUSD()
	}

	var consolidatedWorkloads []*entities.ConsolidatedWorkloadSnapshotModel
	for key, stats := range grouped {
		count := float64(stats.count)

		var cpuLimit *float64
		if stats.cpuLimitCount > 0 {
			val := stats.cpuLimitSum / float64(stats.cpuLimitCount)
			cpuLimit = &val
		}
		var memLimit *float64
		if stats.memLimitCount > 0 {
			val := stats.memLimitSum / float64(stats.memLimitCount)
			memLimit = &val
		}

		wCPUReqAvg := stats.cpuReqSum / count
		wCPUUsedAvg := stats.cpuUsedSum / count
		wMemReqAvg := stats.memReqSum / count
		wMemUsedAvg := stats.memUsedSum / count

		cw := entities.NewConsolidatedWorkloadSnapshot(
			0,
			key.namespace, key.pod, key.container,
			wCPUReqAvg, wCPUUsedAvg, cpuLimit,
			wMemReqAvg, wMemUsedAvg, memLimit,
			ratioOver(true, wCPUReqAvg-wCPUUsedAvg, wCPUReqAvg),
			ratioOver(true, wMemReqAvg-wMemUsedAvg, wMemReqAvg),
			numeric.Round4(stats.projectedSum/count),
		)
		if memLimit != nil && *memLimit > 0 {
			cw.SetOOMRiskScore(numeric.Round4(wMemUsedAvg / *memLimit))
		}
		consolidatedWorkloads = append(consolidatedWorkloads, cw)
	}

	if err := c.db.SaveConsolidation(ctx, consolidation, consolidatedWorkloads); err != nil {
		telemetry.ConsolidationErrors.Inc()
		c.logger.Error("failed to save consolidated snapshots", "error", err)
		return
	}

	c.logger.Info("consolidation saved",
		"id", consolidation.ID(),
		"workloads", len(consolidatedWorkloads),
		"scrapes", scrapesCount,
		"cpu_waste", cpuWaste,
		"mem_waste", memWaste,
	)

	if c.tracker != nil {
		c.tracker.Observe(ctx, notifications.IndicatorKey{Indicator: "cpu_waste_ratio"}, cpuWaste)
		c.tracker.Observe(ctx, notifications.IndicatorKey{Indicator: "mem_waste_ratio"}, memWaste)
		c.tracker.Observe(ctx, notifications.IndicatorKey{Indicator: "pvc_waste_ratio"}, pvcWaste)
		c.tracker.Observe(ctx, notifications.IndicatorKey{Indicator: "hpa_efficiency"}, hpaEff)

		for _, cw := range consolidatedWorkloads {
			c.tracker.Observe(ctx, notifications.IndicatorKey{
				Indicator: "cpu_waste_ratio",
				Namespace: cw.Namespace(), Pod: cw.Pod(), Container: cw.Container(),
			}, cw.CPUWasteRatio())
			c.tracker.Observe(ctx, notifications.IndicatorKey{
				Indicator: "mem_waste_ratio",
				Namespace: cw.Namespace(), Pod: cw.Pod(), Container: cw.Container(),
			}, cw.MemWasteRatio())
		}
	}
}
