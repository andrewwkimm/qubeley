// Package collectors provides system metric collection implementations.
package collectors

import (
	"fmt"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryCollector collects virtual and swap memory metrics using gopsutil.
type MemoryCollector struct {
	interval time.Duration
	hostname string
}

// NewMemoryCollector creates a new MemoryCollector with the specified interval.
func NewMemoryCollector(interval time.Duration) (*MemoryCollector, error) {
	return &MemoryCollector{
		interval: interval,
		hostname: resolveHostname(),
	}, nil
}

// Name returns the collector identifier.
func (c *MemoryCollector) Name() string {
	return "memory"
}

// Interval returns the collection frequency.
func (c *MemoryCollector) Interval() time.Duration {
	return c.interval
}

// Collect gathers virtual and swap memory metrics.
// Returns an error only if virtual memory stats cannot be obtained,
// as those are the primary metric. Swap failures are non-fatal.
func (c *MemoryCollector) Collect() (models.Metric, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	return &models.MemoryMetrics{
		BaseMetric: models.BaseMetric{
			Timestamp:  time.Now(),
			Hostname:   c.hostname,
			MetricType: c.Name(),
		},
		TotalBytes:     vm.Total,
		AvailableBytes: vm.Available,
		UsedBytes:      vm.Used,
		UsedPercent:    vm.UsedPercent,
		Swap:           c.swapMetrics(),
	}, nil
}

// swapMetrics fetches swap memory statistics.
// Non-fatal: returns a zero-value SwapMetrics on error, as swap may be
// disabled or unavailable on some systems (e.g. containerized environments).
func (c *MemoryCollector) swapMetrics() models.SwapMetrics {
	swap, err := mem.SwapMemory()
	if err != nil {
		return models.SwapMetrics{}
	}

	return models.SwapMetrics{
		TotalBytes:  swap.Total,
		UsedBytes:   swap.Used,
		UsedPercent: swap.UsedPercent,
	}
}
