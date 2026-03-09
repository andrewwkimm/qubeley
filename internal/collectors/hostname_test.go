package collectors

import "testing"

func TestResolveHostname(t *testing.T) {
	t.Parallel()

	got := resolveHostname()
	if got == "" {
		t.Error("resolveHostname() returned empty string, want non-empty")
	}
	// "unknown" is acceptable when os.Hostname fails, but the result
	// must always be a non-empty, usable partition key for Kafka.
}
