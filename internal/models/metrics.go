// Package models defines the shared data types for metrics collected and streamed by Qubeley.
package models

import "time"

// Metric is the common interface all metric types must satisfy.
// It enables type-safe passing through the collector and producer pipeline
// without resorting to interface{}.
type Metric interface {
	GetHostname() string
	Type() string
}

// BaseMetric contains fields common to all metric types.
// It is intended to be embedded in concrete metric structs.
type BaseMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	Hostname   string    `json:"hostname"`
	MetricType string    `json:"metric_type"`
}

// GetHostname returns the hostname for Kafka partition keying.
func (b BaseMetric) GetHostname() string {
	return b.Hostname
}

// Type returns the metric type identifier.
func (b BaseMetric) Type() string {
	return b.MetricType
}

// CPUMetrics represents a single CPU usage observation.
type CPUMetrics struct {
	BaseMetric
	TotalPercent float64            `json:"total_percent"`
	PerCore      []float64          `json:"per_core,omitempty"`
	LoadAverage  [3]float64         `json:"load_average"`
	CoreCount    int                `json:"core_count"`
	Extra        map[string]float64 `json:"extra,omitempty"`
}

// SwapMetrics holds swap space usage statistics.
// All fields are zero if swap is disabled or unavailable.
type SwapMetrics struct {
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// MemoryMetrics represents a single memory usage observation.
type MemoryMetrics struct {
	BaseMetric
	TotalBytes     uint64      `json:"total_bytes"`
	AvailableBytes uint64      `json:"available_bytes"`
	UsedBytes      uint64      `json:"used_bytes"`
	UsedPercent    float64     `json:"used_percent"`
	Swap           SwapMetrics `json:"swap"`
}

// TemperatureReading represents a single sensor's temperature observation.
type TemperatureReading struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature_celsius"`
	High        float64 `json:"high_celsius,omitempty"`
	Critical    float64 `json:"critical_celsius,omitempty"`
}

// TemperatureMetrics represents a snapshot of all available temperature sensors.
type TemperatureMetrics struct {
	BaseMetric
	Readings []TemperatureReading `json:"readings"`
}
