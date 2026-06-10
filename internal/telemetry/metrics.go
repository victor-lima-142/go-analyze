package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ScrapesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "go_analyze_scrapes_total",
		Help: "Number of background scrapes attempted.",
	})

	ScrapesErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "go_analyze_scrape_errors_total",
		Help: "Number of background scrapes that failed.",
	})

	ScrapeDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "go_analyze_scrape_duration_seconds",
		Help:    "Duration of background scrape cycles.",
		Buckets: prometheus.ExponentialBuckets(0.05, 2, 10),
	})

	ConsolidationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "go_analyze_consolidations_total",
		Help: "Number of consolidations executed.",
	})

	ConsolidationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "go_analyze_consolidation_errors_total",
		Help: "Number of consolidations that failed.",
	})

	NotificationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "go_analyze_notifications_total",
		Help: "Number of overprovisioning notifications emitted.",
	}, []string{"indicator"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "go_analyze_http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method", "status"})
)
