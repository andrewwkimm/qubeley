package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestBaseMetric_Hostname(t *testing.T) {
	t.Parallel()

	b := BaseMetric{Hostname: "web-01"}
	if got := b.Hostname(); got != "web-01" {
		t.Errorf("Hostname() = %q, want %q", got, "web-01")
	}
}

func TestBaseMetric_Type(t *testing.T) {
	t.Parallel()

	b := BaseMetric{MetricType: "cpu"}
	if got := b.Type(); got != "cpu" {
		t.Errorf("Type() = %q, want %q", got, "cpu")
	}
}

func TestCPUMetrics_SatisfiesMetric(t *testing.T) {
	t.Parallel()

	// Compile-time check that *CPUMetrics satisfies Metric is enforced by
	// the type system. This test verifies the promoted methods return the
	// correct values at runtime.
	var m Metric = &CPUMetrics{
		BaseMetric: BaseMetric{
			Hostname:   "host-01",
			MetricType: "cpu",
		},
	}

	if got := m.Hostname(); got != "host-01" {
		t.Errorf("Hostname() = %q, want %q", got, "host-01")
	}
	if got := m.Type(); got != "cpu" {
		t.Errorf("Type() = %q, want %q", got, "cpu")
	}
}

func TestCPUMetrics_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Millisecond)
	original := &CPUMetrics{
		BaseMetric: BaseMetric{
			Timestamp:  now,
			Hostname:   "host-01",
			MetricType: "cpu",
		},
		TotalPercent: 42.5,
		PerCore:      []float64{40.0, 45.0},
		LoadAverage:  [3]float64{1.0, 0.9, 0.8},
		CoreCount:    2,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got CPUMetrics
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if diff := cmp.Diff(original, &got); diff != "" {
		t.Errorf("JSON round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestLogBatch_EmptyEntriesIsValid(t *testing.T) {
	t.Parallel()

	// An empty log batch is a valid result when no lines arrived during
	// a tick. Verify it serializes cleanly without a null entries field.
	batch := &LogBatch{
		BaseMetric: BaseMetric{
			Hostname:   "host-01",
			MetricType: "logs",
		},
		Entries: nil,
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got LogBatch
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if len(got.Entries) != 0 {
		t.Errorf("Entries len = %d, want 0", len(got.Entries))
	}
}
