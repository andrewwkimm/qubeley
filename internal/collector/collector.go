// The Collector interface.

package collector

import (
	"time"
)

// The interface all metric collectors must implement
type Collector interface {
	Collect() (interface{}, error)
	Name() string
	Interval() time.Duration
}
