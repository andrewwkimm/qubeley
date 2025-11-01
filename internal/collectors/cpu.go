// Package collectors provides system metric collection implementations.
package collectors

import (
	"fmt"
	"os"
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
	hostname, err := os.Hostname()
	if err != nil {
		// Non-fatal: continue with unknown hostname
		hostname = "unknown"
	}

	return &CPUCollector{
		interval:      interval,
		collectPerCPU: collectPerCPU,
		hostname:      hostname,
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
func (c *CPUCollector) Collect() (interface{}, error) {
	now := time.Now()

	totalPercent, err := c.getTotalPercent()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percent: %w", err)
	}

	perCore := c.getPerCorePercent()
	loadAvg := c.getLoadAverage()
	coreCount := c.getCoreCount()

	metrics := &models.CPUMetrics{
		BaseMetric: models.BaseMetric{
			Timestamp:  now,
			Hostname:   c.hostname,
			MetricType: c.Name(),
		},
		TotalPercent: totalPercent,
		PerCore:      perCore,
		LoadAverage:  loadAvg,
		CoreCount:    coreCount,
	}

	return metrics, nil
}

// getTotalPercent fetches overall CPU usage percentage.
// This is a critical metric - errors are propagated to the caller.
func (c *CPUCollector) getTotalPercent() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU percentage data returned")
	}

	return percentages[0], nil
}

// getPerCorePercent fetches per-core CPU usage if configured.
// Non-fatal: returns nil on error or if per-CPU collection is disabled.
func (c *CPUCollector) getPerCorePercent() []float64 {
	if !c.collectPerCPU {
		return nil
	}

	perCore, err := cpu.Percent(0, true)
	if err != nil {
		// TODO: replace with structured logger in production
		fmt.Printf("warning: failed to get per-CPU stats: %v\n", err)
		return nil
	}

	return perCore
}

// getLoadAverage fetches system load averages (1, 5, and 15 minute).
// Non-fatal: returns zero values on error (e.g., on Windows where load average is unavailable).
func (c *CPUCollector) getLoadAverage() [3]float64 {
	loadAvg, err := load.Avg()
	if err != nil {
		return [3]float64{0, 0, 0}
	}

	return [3]float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
}

// getCoreCount returns the number of logical CPU cores.
// Non-fatal: returns 0 on error (will be obvious in metrics dashboard).
func (c *CPUCollector) getCoreCount() int {
	count, err := cpu.Counts(true)
	if err != nil {
		return 0
	}

	return count
}
