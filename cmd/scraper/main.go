package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victor-lima-142/go-analyze/internal/consolidator"
	"github.com/victor-lima-142/go-analyze/internal/metrics"
	"github.com/victor-lima-142/go-analyze/internal/notifications"
	"github.com/victor-lima-142/go-analyze/internal/observability"
	"github.com/victor-lima-142/go-analyze/internal/prometheus"
	"github.com/victor-lima-142/go-analyze/internal/scraper"
	"github.com/victor-lima-142/go-analyze/internal/telemetry"
	"github.com/victor-lima-142/go-analyze/pkg/config"
	"github.com/victor-lima-142/go-analyze/pkg/database"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := telemetry.Init(rootCtx, "go-analyze-scraper", cfg.OTelEnabled, logger)
	if err != nil {
		logger.Error("otel init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}()

	logger.Info("starting scraper",
		"prometheus", cfg.PrometheusEndpoint,
		"scrape_interval", cfg.ScrapeInterval,
		"consolidate_interval", cfg.ConsolidateInterval,
		"cost_model", cfg.CostModelLabel,
	)

	dbClient, err := database.Connect(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := dbClient.Close(); err != nil {
			logger.Error("failed to close database connection", "error", err)
		}
	}()

	if err := dbClient.Migrate(); err != nil {
		logger.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database migrations completed")

	promClient, err := prometheus.NewHTTPPrometheusClient(cfg.PrometheusEndpoint, cfg.QueryTimeout)
	if err != nil {
		logger.Error("failed to create prometheus client", "error", err)
		os.Exit(1)
	}

	obsFilter := observability.NewFilter(cfg.ObservabilityNS, cfg.ObservabilityPattern)
	calculator := metrics.NewCalculator(metrics.CalculatorOptions{
		Client:             promClient,
		CPUHourlyUSD:       cfg.CPUHourlyUSD,
		MemoryGiBHourlyUSD: cfg.MemoryGiBHourlyUSD,
		MonthlyHours:       cfg.MonthlyHours,
		CostModelLabel:     cfg.CostModelLabel,
		Logger:             logger,
		Filter:             obsFilter,
	})

	notifier := notifications.NewSlogNotifier(logger)
	tracker := notifications.NewTracker(map[string]notifications.IndicatorRule{
		"cpu_waste_ratio": {Threshold: cfg.WasteThreshold, RequiredDuration: cfg.SustainedCPUMemDur, Comparator: notifications.GreaterThan},
		"mem_waste_ratio": {Threshold: cfg.WasteThreshold, RequiredDuration: cfg.SustainedCPUMemDur, Comparator: notifications.GreaterThan},
	}, notifier, cfg.NotificationCooldown, 2*cfg.ConsolidateInterval)

	bgScraper := scraper.NewScraper(calculator, dbClient, cfg.ScrapeInterval, cfg.DefaultWindow, logger)
	go bgScraper.Start(rootCtx)

	bgConsolidator := consolidator.NewConsolidator(dbClient, cfg.ConsolidateInterval, cfg.ConsolidateWindow, logger, tracker)
	go bgConsolidator.Start(rootCtx)

	<-rootCtx.Done()
	logger.Info("scraper stopped")
}
