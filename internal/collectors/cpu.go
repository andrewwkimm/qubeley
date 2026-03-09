// Package collectors provides system metric collection implementations.
package collectors

import (
	"fmt"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
)

// CPUCollector collects CPU metrics using gopsutil.
type CPUCollector struct {
	interval      time.Duration
	collectPerCPU bool
	hostname      string
}

// NewCPUCollector creates a new CPU collector with the specified configuration.
// interval specifies how often to collect metrics.
// collectPerCPU determines whether to collect per-core statistics.
func NewCPUCollector(interval time.Duration, collectPerCPU bool) (*CPUCollector, error) {
	return &CPUCollector{
		interval:      interval,
		collectPerCPU: collectPerCPU,
		hostname:      resolveHostname(),
	}, nil
}

// Name returns the collector identifier.
func (c *CPUCollector) Name() string {
	return "cpu"
}

// Interval returns the collection frequency.
func (c *CPUCollector) Interval() time.Duration {
	return c.interval
}

// Collect gathers CPU metrics and returns them.
// Returns an error only if critical metrics (total CPU percentage) cannot be obtained.
func (c *CPUCollector) Collect() (models.Metric, error) {
	now := time.Now()

	total, err := c.totalPercent()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percent: %w", err)
	}

	metrics := &models.CPUMetrics{
		BaseMetric: models.BaseMetric{
			Timestamp:  now,
			Hostname:   c.hostname,
			MetricType: c.Name(),
		},
		TotalPercent: total,
		PerCore:      c.perCorePercent(),
		LoadAverage:  c.loadAverage(),
		CoreCount:    c.coreCount(),
	}

	return metrics, nil
}

// totalPercent fetches overall CPU usage percentage.
// This is a critical metric — errors are propagated to the caller.
func (c *CPUCollector) totalPercent() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU percentage data returned")
	}

	return percentages[0], nil
}

// perCorePercent fetches per-core CPU usage if configured.
// Non-fatal: returns nil on error or if per-CPU collection is disabled.
func (c *CPUCollector) perCorePercent() []float64 {
	if !c.collectPerCPU {
		return nil
	}

	perCore, err := cpu.Percent(0, true)
	if err != nil {
		return nil
	}

	return perCore
}

// loadAverage fetches system load averages (1, 5, and 15 minute).
// Non-fatal: returns zero values on error (e.g., on Windows where load average is unavailable).
func (c *CPUCollector) loadAverage() [3]float64 {
	avg, err := load.Avg()
	if err != nil {
		return [3]float64{}
	}

	return [3]float64{avg.Load1, avg.Load5, avg.Load15}
}

// coreCount returns the number of logical CPU cores.
// Non-fatal: returns 0 on error (will be obvious in metrics dashboard).
func (c *CPUCollector) coreCount() int {
	count, err := cpu.Counts(true)
	if err != nil {
		return 0
	}

	return count
}
