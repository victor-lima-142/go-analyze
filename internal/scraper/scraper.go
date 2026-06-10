package scraper

import (
	"context"
	"log/slog"
	"time"

	"go-analyze/internal/metrics"
	"go-analyze/internal/telemetry"
)

type Scraper struct {
	calculator    *metrics.Calculator
	db            SnapshotSaver
	interval      time.Duration
	defaultWindow string
	logger        *slog.Logger
}

func NewScraper(calculator *metrics.Calculator, db SnapshotSaver, interval time.Duration, defaultWindow string, logger *slog.Logger) *Scraper {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scraper{
		calculator:    calculator,
		db:            db,
		interval:      interval,
		defaultWindow: defaultWindow,
		logger:        logger,
	}
}

func (s *Scraper) Start(ctx context.Context) {
	s.logger.Info("starting background scraper", "interval", s.interval, "default_window", s.defaultWindow)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.scrapeAndSave(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping background scraper: context cancelled")
			return
		case <-ticker.C:
			s.scrapeAndSave(ctx)
		}
	}
}

func (s *Scraper) scrapeAndSave(ctx context.Context) {
	telemetry.ScrapesTotal.Inc()
	tracer := telemetry.Tracer("scraper")
	ctx, span := tracer.Start(ctx, "scrape_cycle")
	defer span.End()

	started := time.Now()

	res, err := s.calculator.Calculate(ctx, metrics.QueryParams{Window: s.defaultWindow})
	if err != nil {
		telemetry.ScrapesErrors.Inc()
		s.logger.Error("background scrape failed on calculation", "error", err)
		return
	}

	duration := time.Since(started)
	telemetry.ScrapeDuration.Observe(duration.Seconds())
	s.logger.Debug("scrape calculation finished", "duration", duration)

	if err := s.db.SaveSnapshot(ctx, res, duration); err != nil {
		telemetry.ScrapesErrors.Inc()
		s.logger.Error("background scrape failed to persist snapshot", "error", err)
		return
	}
	s.logger.Info("scrape snapshot persisted", "items", len(res.Items), "duration", duration)
}
