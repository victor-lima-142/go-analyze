package prometheus

import (
	"context"
	"time"

	"go-analyze/pkg/models"
)

type MetricPoint = models.MetricPoint
type MetricSeries = models.QueryResult

type PrometheusClient interface {
	Query(ctx context.Context, promql string, ts time.Time) ([]models.QueryResult, error)
	QueryRange(ctx context.Context, promql string, r QueryRange) ([]models.QueryResult, error)
}

type QueryRange struct {
	Start time.Time
	End   time.Time
	Step  time.Duration
}

type QueryFilter struct {
	Namespace string
	Pod       string
	Container string
	PVC       string
	HPA       string
}

type IndicatorInputDataset struct {
	CPURequestsBytesOrCores []models.QueryResult `json:"cpuRequests"`
	CPUUsageCores           []models.QueryResult `json:"cpuUsage"`
	MemoryRequestsBytes     []models.QueryResult `json:"memoryRequestsBytes"`
	MemoryUsageBytes        []models.QueryResult `json:"memoryUsageBytes"`
	PVCCapacityBytes        []models.QueryResult `json:"pvcCapacityBytes"`
	PVCUsedBytes            []models.QueryResult `json:"pvcUsedBytes"`
	HPACurrentReplicas      []models.QueryResult `json:"hpaCurrentReplicas"`
	HPADesiredReplicas      []models.QueryResult `json:"hpaDesiredReplicas"`
	HPAReplicaSpec          []models.QueryResult `json:"hpaReplicaSpec"`
}
