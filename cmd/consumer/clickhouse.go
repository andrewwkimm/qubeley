package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andrewwkimm/qubeley/internal/models"
)

// clickHouseClient sends data to ClickHouse via the HTTP interface.
// This avoids any native driver dependency — ClickHouse's HTTP API accepts
// INSERT statements with tab-separated or JSON values directly.
type clickHouseClient struct {
	cfg    config
	client *http.Client
}

func newClickHouseClient(cfg config) (*clickHouseClient, error) {
	ch := &clickHouseClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	// Verify connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ch.ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping failed: %w", err)
	}
	return ch, nil
}

func (c *clickHouseClient) close() {}

// ping sends a lightweight SELECT 1 to verify the connection.
func (c *clickHouseClient) ping(ctx context.Context) error {
	_, err := c.query(ctx, "SELECT 1")
	return err
}

// query executes a raw SQL string and returns the response body.
func (c *clickHouseClient) query(ctx context.Context, sql string) (string, error) {
	params := url.Values{}
	params.Set("query", sql)
	params.Set("database", c.cfg.chDB)
	params.Set("user", c.cfg.chUser)
	params.Set("password", c.cfg.chPassword)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.chURL+"/?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("clickhouse error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return string(body), nil
}

// exec sends an INSERT statement with a VALUES body via POST.
// The query parameter holds the INSERT ... FORMAT Values header;
// the body holds the actual rows.
func (c *clickHouseClient) exec(ctx context.Context, insertSQL string, body string) error {
	params := url.Values{}
	params.Set("query", insertSQL)
	params.Set("database", c.cfg.chDB)
	params.Set("user", c.cfg.chUser)
	params.Set("password", c.cfg.chPassword)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.chURL+"/?"+params.Encode(),
		strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clickhouse error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

// ts formats a time.Time as the string ClickHouse expects for DateTime64(3).
func ts(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000")
}

// esc escapes a string value for use inside single quotes in ClickHouse SQL.
func esc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

func (c *clickHouseClient) insertCPU(ctx context.Context, m *models.CPUMetrics) error {
	sql := `INSERT INTO cpu_metrics
		(timestamp, hostname, total_percent, core_count, load_1m, load_5m, load_15m)
		FORMAT Values`

	row := fmt.Sprintf("('%s','%s',%f,%d,%f,%f,%f)",
		ts(m.Timestamp),
		esc(m.Hostname),
		m.TotalPercent,
		m.CoreCount,
		m.LoadAverage[0],
		m.LoadAverage[1],
		m.LoadAverage[2],
	)

	return c.exec(ctx, sql, row)
}

func (c *clickHouseClient) insertMemory(ctx context.Context, m *models.MemoryMetrics) error {
	sql := `INSERT INTO memory_metrics
		(timestamp, hostname, total_bytes, available_bytes, used_bytes, used_percent,
		 swap_total_bytes, swap_used_bytes, swap_used_percent)
		FORMAT Values`

	row := fmt.Sprintf("('%s','%s',%d,%d,%d,%f,%d,%d,%f)",
		ts(m.Timestamp),
		esc(m.Hostname),
		m.TotalBytes,
		m.AvailableBytes,
		m.UsedBytes,
		m.UsedPercent,
		m.Swap.TotalBytes,
		m.Swap.UsedBytes,
		m.Swap.UsedPercent,
	)

	return c.exec(ctx, sql, row)
}

func (c *clickHouseClient) insertTemperature(ctx context.Context, m *models.TemperatureMetrics) error {
	if len(m.Readings) == 0 {
		return nil
	}

	sql := `INSERT INTO temperature_metrics
		(timestamp, hostname, sensor_key, temperature_celsius, high_celsius, critical_celsius)
		FORMAT Values`

	rows := make([]string, 0, len(m.Readings))
	for _, r := range m.Readings {
		rows = append(rows, fmt.Sprintf("('%s','%s','%s',%f,%f,%f)",
			ts(m.Timestamp),
			esc(m.Hostname),
			esc(r.SensorKey),
			r.Temperature,
			r.High,
			r.Critical,
		))
	}

	return c.exec(ctx, sql, strings.Join(rows, ","))
}

func (c *clickHouseClient) insertLogs(ctx context.Context, m *models.LogBatch) error {
	if len(m.Entries) == 0 {
		return nil
	}

	sql := `INSERT INTO logs (timestamp, hostname, unit, priority, message) FORMAT Values`

	rows := make([]string, 0, len(m.Entries))
	for _, e := range m.Entries {
		rows = append(rows, fmt.Sprintf("('%s','%s','%s',%d,'%s')",
			ts(e.Timestamp),
			esc(m.Hostname),
			esc(e.Unit),
			e.Priority,
			esc(e.Message),
		))
	}

	return c.exec(ctx, sql, strings.Join(rows, ","))
}

// insertRaw writes the full JSON payload to the raw_metrics audit table.
func (c *clickHouseClient) insertRaw(ctx context.Context, base *models.BaseMetric, payload string) error {
	sql := `INSERT INTO raw_metrics (timestamp, hostname, metric_type, payload) FORMAT Values`

	row := fmt.Sprintf("('%s','%s','%s','%s')",
		ts(base.Timestamp),
		esc(base.Hostname),
		esc(base.MetricType),
		esc(payload),
	)

	return c.exec(ctx, sql, row)
}