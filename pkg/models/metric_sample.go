package models

type MetricSample struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}
