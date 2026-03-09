// The Collector interface.

package collectors

import (
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
)

// Collector defines the interface that all metric collectors must implement.
type Collector interface {
	Collect() (models.Metric, error)
	Name() string
	Interval() time.Duration
}
