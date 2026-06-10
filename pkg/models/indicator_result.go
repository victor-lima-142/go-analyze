package models

type QueryParams struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	PVC       string `json:"pvc"`
	HPA       string `json:"hpa"`
	Window    string `json:"window"`
}

type Indicators struct {
	CPUWasteRatio            float64 `json:"cpu_waste_ratio"`
	MemWasteRatio            float64 `json:"mem_waste_ratio"`
	PVCWasteRatio            float64 `json:"pvc_waste_ratio"`
	HPAEfficiency            float64 `json:"hpa_efficiency"`
	ProjectedMonthlyWasteUSD float64 `json:"projected_monthly_waste_usd"`
	OOMRiskScore             float64 `json:"oom_risk_score"`
}

type Inputs struct {
	CPURequestedCores    float64 `json:"cpu_requested_cores"`
	CPUUsedCores         float64 `json:"cpu_used_cores"`
	CPULimitCores        float64 `json:"cpu_limit_cores"`
	MemoryRequestedBytes float64 `json:"memory_requested_bytes"`
	MemoryUsedBytes      float64 `json:"memory_used_bytes"`
	MemoryLimitBytes     float64 `json:"memory_limit_bytes"`
	HPAAvgReplicas       float64 `json:"hpa_avg_replicas"`
	HPAMaxReplicas       float64 `json:"hpa_max_replicas"`
	PVCCapacityBytes     float64 `json:"pvc_capacity_bytes"`
	PVCUsedBytes         float64 `json:"pvc_used_bytes"`
}

type IndicatorResult struct {
	Filters    QueryParams    `json:"filters"`
	Indicators Indicators     `json:"indicators"`
	Inputs     Inputs         `json:"inputs"`
	Items      []WorkloadItem `json:"items,omitempty"`
	CostModel  string         `json:"cost_model,omitempty"`
}

type WorkloadItem struct {
	Namespace                string  `json:"namespace"`
	Pod                      string  `json:"pod"`
	Container                string  `json:"container"`
	CPURequestedCores        float64 `json:"cpu_requested_cores"`
	CPUUsedCores             float64 `json:"cpu_used_cores"`
	CPULimitCores            float64 `json:"cpu_limit_cores,omitempty"`
	MemoryRequestedBytes     float64 `json:"memory_requested_bytes"`
	MemoryUsedBytes          float64 `json:"memory_used_bytes"`
	MemoryLimitBytes         float64 `json:"memory_limit_bytes,omitempty"`
	CPUWasteRatio            float64 `json:"cpu_waste_ratio"`
	MemWasteRatio            float64 `json:"mem_waste_ratio"`
	OOMRiskScore             float64 `json:"oom_risk_score"`
	ProjectedMonthlyWasteUSD float64 `json:"projected_monthly_waste_usd"`
}
