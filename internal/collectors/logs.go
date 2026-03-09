// Package collectors provides system metric collection implementations.
package collectors

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
)

// ErrNoJournal is returned when journalctl is not available on the system.
// Callers may use errors.Is to distinguish this from other failures.
var ErrNoJournal = errors.New("journalctl not found: system may not use systemd")

// journalEntry is the raw JSON structure emitted by journalctl --output=json.
// Only the fields Qubeley uses are mapped; the rest are silently ignored.
type journalEntry struct {
	RealtimeTimestamp string `json:"__REALTIME_TIMESTAMP"`
	SystemdUnit       string `json:"_SYSTEMD_UNIT"`
	Priority          string `json:"PRIORITY"`
	Message           string `json:"MESSAGE"`
}

// LogsCollector tails the systemd journal via a journalctl subprocess and
// buffers log entries between calls to Collect.
//
// The subprocess is started on the first call to Collect and runs for the
// lifetime of the collector. Entries are accumulated in a mutex-protected
// buffer and drained on each tick.
type LogsCollector struct {
	interval time.Duration
	hostname string

	mu      sync.Mutex
	buf     []models.LogEntry
	started bool
	err     error // sticky error from the reader goroutine
}

// NewLogsCollector creates a new LogsCollector with the specified interval.
// It does not start the journalctl subprocess; that happens on the first
// call to Collect.
func NewLogsCollector(interval time.Duration) (*LogsCollector, error) {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return nil, ErrNoJournal
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &LogsCollector{
		interval: interval,
		hostname: resolveHostname(),
	}, nil
}

// Name returns the collector identifier.
func (c *LogsCollector) Name() string {
	return "logs"
}

// Interval returns the collection frequency.
func (c *LogsCollector) Interval() time.Duration {
	return c.interval
}

// Collect drains all log entries buffered since the last call and returns
// them as a LogBatch. An empty Entries slice is valid and indicates no new
// lines arrived during this tick.
//
// The journalctl subprocess is started on the first call. If the subprocess
// exits unexpectedly, subsequent calls return the error.
func (c *LogsCollector) Collect() (models.Metric, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		c.started = true
		go c.stream()
	}

	if c.err != nil {
		return nil, c.err
	}

	entries := c.buf
	c.buf = nil

	return &models.LogBatch{
		BaseMetric: models.BaseMetric{
			Timestamp:  time.Now(),
			Hostname:   c.hostname,
			MetricType: c.Name(),
		},
		Entries: entries,
	}, nil
}

// stream runs in a background goroutine and reads JSON lines from journalctl
// into the buffer. It sets c.err if the subprocess exits or stdout cannot
// be read, which causes subsequent Collect calls to surface the error.
func (c *LogsCollector) stream() {
	cmd := exec.Command(
		"journalctl",
		"--follow",
		"--output=json",
		"--no-pager",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.setErr(fmt.Errorf("failed to open journalctl stdout pipe: %w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		c.setErr(fmt.Errorf("failed to start journalctl: %w", err))
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		entry, err := parseJournalLine(scanner.Bytes())
		if err != nil {
			// Malformed lines are skipped rather than fatal; journald
			// occasionally emits non-JSON lines during cursor seeks.
			continue
		}

		c.mu.Lock()
		c.buf = append(c.buf, entry)
		c.mu.Unlock()
	}

	if err := cmd.Wait(); err != nil {
		c.setErr(fmt.Errorf("journalctl exited unexpectedly: %w", err))
	}
}

// setErr stores a sticky error under the lock.
func (c *LogsCollector) setErr(err error) {
	c.mu.Lock()
	c.err = err
	c.mu.Unlock()
}

// parseJournalLine parses a single JSON line from journalctl --output=json
// into a LogEntry. Microsecond timestamps from __REALTIME_TIMESTAMP are
// converted to time.Time.
func parseJournalLine(raw []byte) (models.LogEntry, error) {
	var je journalEntry
	if err := json.Unmarshal(raw, &je); err != nil {
		return models.LogEntry{}, fmt.Errorf("failed to unmarshal journal entry: %w", err)
	}

	ts, err := parseJournalTimestamp(je.RealtimeTimestamp)
	if err != nil {
		ts = time.Now() // best-effort fallback
	}

	priority, _ := strconv.Atoi(je.Priority) // zero is a valid fallback (Emergency)

	return models.LogEntry{
		Timestamp: ts,
		Unit:      je.SystemdUnit,
		Priority:  priority,
		Message:   je.Message,
	}, nil
}

// parseJournalTimestamp converts journald's __REALTIME_TIMESTAMP, which is
// microseconds since the Unix epoch as a decimal string, to a time.Time.
func parseJournalTimestamp(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	us, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", raw, err)
	}

	return time.Unix(us/1e6, (us%1e6)*1e3), nil
}
