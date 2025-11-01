// Metrics data models

package models

import "time"

// BaseMetric to define common fields for all metrics
type BaseMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	Hostname   string    `json:"hostname"`
	MetricType string    `json:"metric_type"`
}

// CPU usage data
type CPUMetrics struct {
	BaseMetric
	TotalPercent float64            `json:"total_percent"`
	PerCore      []float64          `json:"per_core,omitempty"`
	LoadAverage  [3]float64         `json:"load_average"`
	CoreCount    int                `json:"core_count"`
	Extra        map[string]float64 `json:"extra,omitempty"` // For future metrics
}
