// The Collector interface.

package collector

import "time"

type Metric struct {
	Timestamp time.Time
	Source    string
	Data      interface{} // Will be your specific metric struct
}

type Collector interface {
	Collect() (*Metric, error)
	Name() string
}
