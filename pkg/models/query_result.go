package models

import "time"

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type QueryResult struct {
	MetricName string            `json:"metricName"`
	Labels     map[string]string `json:"labels"`
	Points     []MetricPoint     `json:"points"`
}
