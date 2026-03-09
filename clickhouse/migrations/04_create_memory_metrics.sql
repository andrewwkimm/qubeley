-- Structured memory metrics written by Flink after processing raw_metrics.
CREATE TABLE IF NOT EXISTS qubeley.memory_metrics
(
    timestamp         DateTime64(3),
    hostname          LowCardinality(String),
    total_bytes       UInt64,
    available_bytes   UInt64,
    used_bytes        UInt64,
    used_percent      Float64,
    swap_total_bytes  UInt64,
    swap_used_bytes   UInt64,
    swap_used_percent Float64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (hostname, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
