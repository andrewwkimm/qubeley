// The Collector interface.

package collector

import (
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
)

// The interface all metric collectors must implement
type Collector interface {
	Collect() (*models.BaseMetric, error)
	Name() string
	Interval() time.Duration
}
