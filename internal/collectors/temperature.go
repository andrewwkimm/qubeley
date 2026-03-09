// Package collectors provides system metric collection implementations.
package collectors

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
	"github.com/shirou/gopsutil/v3/host"
)

// ErrNoSensors is returned by Collect when the system exposes no temperature
// sensors. Callers may use errors.Is to distinguish this from other failures.
var ErrNoSensors = errors.New("no temperature sensors available")

// TemperatureCollector collects hardware temperature sensor data using gopsutil.
//
// Sensor availability is platform-dependent. On Linux, readings come from
// sysfs; on macOS, from IOKit. Windows is not supported by the underlying
// library. If no sensors are found, Collect returns ErrNoSensors.
type TemperatureCollector struct {
	interval time.Duration
	hostname string
}

// NewTemperatureCollector creates a new TemperatureCollector with the
// specified collection interval.
func NewTemperatureCollector(interval time.Duration) (*TemperatureCollector, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &TemperatureCollector{
		interval: interval,
		hostname: resolveHostname(),
	}, nil
}

// Name returns the collector identifier.
func (c *TemperatureCollector) Name() string {
	return "temperature"
}

// Interval returns the collection frequency.
func (c *TemperatureCollector) Interval() time.Duration {
	return c.interval
}

// Collect gathers temperature readings from all available hardware sensors.
// Returns ErrNoSensors if the system exposes no sensors, which callers can
// treat as a signal to skip or deregister this collector.
func (c *TemperatureCollector) Collect() (models.Metric, error) {
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return nil, fmt.Errorf("failed to read temperature sensors: %w", err)
	}

	if len(sensors) == 0 {
		return nil, ErrNoSensors
	}

	readings := make([]models.TemperatureReading, 0, len(sensors))
	for _, s := range sensors {
		readings = append(readings, models.TemperatureReading{
			SensorKey:   s.SensorKey,
			Temperature: s.Temperature,
			High:        s.High,
			Critical:    s.Critical,
		})
	}

	return &models.TemperatureMetrics{
		BaseMetric: models.BaseMetric{
			Timestamp:  time.Now(),
			Hostname:   c.hostname,
			MetricType: c.Name(),
		},
		Readings: readings,
	}, nil
}
