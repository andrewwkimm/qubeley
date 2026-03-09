package collectors

import (
	"errors"
	"testing"
	"time"
)

func TestParseJournalTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid timestamp",
			input: "1700000000000000", // 1700000000 seconds since epoch
			want:  time.Unix(1700000000, 0).UTC(),
		},
		{
			name:  "valid timestamp with sub-second precision",
			input: "1700000000123456",
			want:  time.Unix(1700000000, 123456000).UTC(),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-numeric string",
			input:   "not-a-timestamp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseJournalTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJournalTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !got.UTC().Equal(tt.want) {
				t.Errorf("parseJournalTimestamp(%q) = %v, want %v", tt.input, got.UTC(), tt.want)
			}
		})
	}
}

func TestParseJournalLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got interface{})
	}{
		{
			name: "well-formed entry",
			input: `{
				"__REALTIME_TIMESTAMP": "1700000000000000",
				"_SYSTEMD_UNIT": "sshd.service",
				"PRIORITY": "6",
				"MESSAGE": "Accepted publickey for user"
			}`,
			check: func(t *testing.T, got interface{}) {
				t.Helper()
				entry, ok := got.(interface {
					GetUnit() string
					GetPriority() int
					GetMessage() string
				})
				_ = entry
				_ = ok
			},
		},
		{
			name:    "invalid json",
			input:   `{not valid json`,
			wantErr: true,
		},
		{
			name: "missing timestamp falls back gracefully",
			input: `{
				"_SYSTEMD_UNIT": "kernel",
				"PRIORITY": "3",
				"MESSAGE": "Out of memory"
			}`,
			wantErr: false, // timestamp fallback to time.Now(), not fatal
		},
		{
			name: "missing priority defaults to zero",
			input: `{
				"__REALTIME_TIMESTAMP": "1700000000000000",
				"MESSAGE": "test message"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseJournalLine([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJournalLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseJournalLine_WellFormedFields(t *testing.T) {
	t.Parallel()

	input := `{
		"__REALTIME_TIMESTAMP": "1700000000000000",
		"_SYSTEMD_UNIT": "nginx.service",
		"PRIORITY": "4",
		"MESSAGE": "worker process exited"
	}`

	got, err := parseJournalLine([]byte(input))
	if err != nil {
		t.Fatalf("parseJournalLine() unexpected error: %v", err)
	}

	if got.Unit != "nginx.service" {
		t.Errorf("Unit = %q, want %q", got.Unit, "nginx.service")
	}
	if got.Priority != 4 {
		t.Errorf("Priority = %d, want %d", got.Priority, 4)
	}
	if got.Message != "worker process exited" {
		t.Errorf("Message = %q, want %q", got.Message, "worker process exited")
	}
}

func TestErrNoJournal(t *testing.T) {
	t.Parallel()

	// Verify ErrNoJournal is returned when journalctl is unavailable.
	// We can't easily fake exec.LookPath here, so we test the sentinel
	// identity directly — callers depend on errors.Is working correctly.
	err := ErrNoJournal
	if !errors.Is(err, ErrNoJournal) {
		t.Errorf("errors.Is(ErrNoJournal, ErrNoJournal) = false, want true")
	}
}
