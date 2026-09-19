package models

type QueryParams struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Window    string `json:"window"`
}

type Indicators struct {
	CPUWasteRatio                  float64 `json:"cpu_waste_ratio"`
	MemWasteRatio                  float64 `json:"mem_waste_ratio"`
	CPUProjectedMonthlyWasteUSD    float64 `json:"cpu_projected_monthly_waste_usd"`
	MemoryProjectedMonthlyWasteUSD float64 `json:"memory_projected_monthly_waste_usd"`
	ProjectedMonthlyWasteUSD       float64 `json:"projected_monthly_waste_usd"`
}

type Inputs struct {
	CPURequestedCores    float64 `json:"cpu_requested_cores"`
	CPUUsedCores         float64 `json:"cpu_used_cores"`
	CPULimitCores        float64 `json:"cpu_limit_cores"`
	MemoryRequestedBytes float64 `json:"memory_requested_bytes"`
	MemoryUsedBytes      float64 `json:"memory_used_bytes"`
	MemoryLimitBytes     float64 `json:"memory_limit_bytes"`
}

type IndicatorResult struct {
	Filters    QueryParams    `json:"filters"`
	Indicators Indicators     `json:"indicators"`
	Inputs     Inputs         `json:"inputs"`
	Items      []WorkloadItem `json:"items,omitempty"`
	CostModel  string         `json:"cost_model,omitempty"`
}

type WorkloadItem struct {
	Namespace                      string  `json:"namespace"`
	Pod                            string  `json:"pod"`
	Container                      string  `json:"container"`
	CPURequestedCores              float64 `json:"cpu_requested_cores"`
	CPUUsedCores                   float64 `json:"cpu_used_cores"`
	CPULimitCores                  float64 `json:"cpu_limit_cores,omitempty"`
	MemoryRequestedBytes           float64 `json:"memory_requested_bytes"`
	MemoryUsedBytes                float64 `json:"memory_used_bytes"`
	MemoryLimitBytes               float64 `json:"memory_limit_bytes,omitempty"`
	CPUWasteRatio                  float64 `json:"cpu_waste_ratio"`
	MemWasteRatio                  float64 `json:"mem_waste_ratio"`
	CPUProjectedMonthlyWasteUSD    float64 `json:"cpu_projected_monthly_waste_usd"`
	MemoryProjectedMonthlyWasteUSD float64 `json:"memory_projected_monthly_waste_usd"`
	ProjectedMonthlyWasteUSD       float64 `json:"projected_monthly_waste_usd"`
}
