package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PrometheusEndpoint   string
	ServerAddress        string
	QueryTimeout         time.Duration
	DefaultWindow        string
	CPUHourlyUSD         float64
	MemoryGiBHourlyUSD   float64
	MonthlyHours         float64
	ScrapeInterval       time.Duration
	ConsolidateInterval  time.Duration
	ConsolidateWindow    time.Duration
	HPAWindow            time.Duration
	PVCWindow            time.Duration
	WasteThreshold       float64
	HPAEfficiencyMin     float64
	SustainedCPUMemDur   time.Duration
	SustainedPVCDur      time.Duration
	NotificationCooldown time.Duration
	ObservabilityNS      []string
	ObservabilityPattern []string
	CostModelLabel       string
	OTelEnabled          bool
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	DBSSLMode            string
}

func Load() (*Config, error) {
	cfg := &Config{
		PrometheusEndpoint:   os.Getenv("PROMETHEUS_ENDPOINT"),
		ServerAddress:        getenv("SERVER_ADDRESS", ":8080"),
		QueryTimeout:         durationEnv("QUERY_TIMEOUT", 10*time.Second),
		DefaultWindow:        getenv("DEFAULT_WINDOW", "10m"),
		CPUHourlyUSD:         floatEnv("CPU_HOURLY_USD", 0.04048),
		MemoryGiBHourlyUSD:   floatEnv("MEMORY_GIB_HOURLY_USD", 0.004445),
		MonthlyHours:         floatEnv("MONTHLY_HOURS", 720),
		ScrapeInterval:       durationEnv("SCRAPE_INTERVAL", 30*time.Second),
		ConsolidateInterval:  durationEnv("CONSOLIDATE_INTERVAL", 1*time.Minute),
		ConsolidateWindow:    durationEnv("CONSOLIDATE_WINDOW", 10*time.Minute),
		HPAWindow:            durationEnv("HPA_WINDOW", 72*time.Hour),
		PVCWindow:            durationEnv("PVC_WINDOW", 7*24*time.Hour),
		WasteThreshold:       floatEnv("WASTE_THRESHOLD", 0.50),
		HPAEfficiencyMin:     floatEnv("HPA_EFFICIENCY_MIN", 0.30),
		SustainedCPUMemDur:   durationEnv("SUSTAINED_CPU_MEM_DURATION", 72*time.Hour),
		SustainedPVCDur:      durationEnv("SUSTAINED_PVC_DURATION", 7*24*time.Hour),
		NotificationCooldown: durationEnv("NOTIFICATION_COOLDOWN", 30*time.Minute),
		ObservabilityNS:      csvEnv("OBSERVABILITY_NAMESPACES", []string{"kube-system", "kubernetes-dashboard"}),
		ObservabilityPattern: csvEnv("OBSERVABILITY_PATTERNS", []string{"prometheus", "kube-state-metrics", "node-exporter", "pushgateway", "configmap-reload", "alertmanager"}),
		CostModelLabel:       getenv("COST_MODEL_LABEL", "aws-fargate-us-east-1-static"),
		OTelEnabled:          boolEnv("OTEL_ENABLED", false),
		DBHost:               getenv("DB_HOST", "localhost"),
		DBPort:               getenv("DB_PORT", "5432"),
		DBUser:               getenv("DB_USER", "postgres"),
		DBPassword:           getenv("DB_PASSWORD", "postgres"),
		DBName:               getenv("DB_NAME", "go_analyze"),
		DBSSLMode:            getenv("DB_SSLMODE", "disable"),
	}

	if cfg.PrometheusEndpoint == "" {
		return nil, errors.New("PROMETHEUS_ENDPOINT is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func floatEnv(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return fallback
}

func csvEnv(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
