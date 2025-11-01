// The Collector interface.

package collectors

import (
	"time"
)

// Collector defines the interface that all metric collectors must implement.
type Collector interface {
	Collect() (interface{}, error)
	Name() string
	Interval() time.Duration
}
