package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/victor-lima-142/go-analyze/docs"
	"github.com/victor-lima-142/go-analyze/internal/consolidator"
	"github.com/victor-lima-142/go-analyze/internal/handlers"
	"github.com/victor-lima-142/go-analyze/internal/metrics"
	"github.com/victor-lima-142/go-analyze/internal/notifications"
	"github.com/victor-lima-142/go-analyze/internal/observability"
	"github.com/victor-lima-142/go-analyze/internal/prometheus"
	"github.com/victor-lima-142/go-analyze/internal/scraper"
	"github.com/victor-lima-142/go-analyze/internal/telemetry"
	"github.com/victor-lima-142/go-analyze/pkg/config"
	"github.com/victor-lima-142/go-analyze/pkg/database"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Go Analyze API
// @version         1.0
// @description     Esta é a API do analisador de recursos e custos de workloads Kubernetes (go-analyze).
// @contact.name    Suporte Go-Analyze
// @host            localhost:8080
// @BasePath        /
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

	otelShutdown, err := telemetry.Init(rootCtx, "go-analyze", cfg.OTelEnabled, logger)
	if err != nil {
		logger.Error("otel init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}()

	logger.Info("starting resource analyzer",
		"prometheus", cfg.PrometheusEndpoint,
		"http", cfg.ServerAddress,
		"default_window", cfg.DefaultWindow,
		"scrape_interval", cfg.ScrapeInterval,
		"cost_model", cfg.CostModelLabel,
		"otel_enabled", cfg.OTelEnabled,
	)

	dbClient, err := database.Connect(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		logger.Info("closing database connection")
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

	indicatorHandler := handlers.NewIndicatorHandler(calculator, cfg.DefaultWindow, logger)
	consolidatedHandler := handlers.NewConsolidatedHandler(dbClient, logger)
	workloadDetailsHandler := handlers.NewWorkloadDetailsHandler(calculator, dbClient, cfg.DefaultWindow, logger)
	auditHandler := handlers.NewAuditHandler(dbClient, cfg.CPUHourlyUSD, cfg.MemoryGiBHourlyUSD, cfg.MonthlyHours, cfg.CostModelLabel, logger)
	experimentResetHandler := handlers.NewExperimentResetHandler(dbClient, tracker, cfg.ExperimentResetEnabled, cfg.ExperimentResetToken, logger)

	healthHandler := handlers.NewHealthHandler(dbClient, func(ctx context.Context) error {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_, err := promClient.QueryInstant(probeCtx, "up")
		return err
	}, logger)

	mux := http.NewServeMux()
	mux.Handle("/healthz", healthHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	mux.Handle("/api/v1/indicators", indicatorHandler)
	mux.Handle("/api/v1/consolidated", consolidatedHandler)
	mux.Handle("/api/v1/workloads/details", workloadDetailsHandler)
	mux.Handle("/api/v1/audit", auditHandler)
	mux.Handle("/api/v1/experiment/reset", experimentResetHandler)

	server := &http.Server{
		Addr:              cfg.ServerAddress,
		Handler:           requestLogMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.ServerAddress)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(s int) {
	r.status = s
	r.ResponseWriter.WriteHeader(s)
}

func requestLogMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		started := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(started)
		telemetry.HTTPRequestDuration.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(rec.status)).Observe(duration.Seconds())
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", duration.Milliseconds(),
		)
	})
}
