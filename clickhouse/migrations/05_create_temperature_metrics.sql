-- Structured temperature metrics written by Flink after processing raw_metrics.
-- Each row represents a single sensor reading; multiple rows share a timestamp
-- when a host reports several sensors in the same collection tick.
CREATE TABLE IF NOT EXISTS qubeley.temperature_metrics
(
    timestamp           DateTime64(3),
    hostname            LowCardinality(String),
    sensor_key          String,
    temperature_celsius Float64,
    high_celsius        Float64,
    critical_celsius    Float64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (hostname, sensor_key, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
