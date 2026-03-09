-- Structured CPU metrics written by Flink after processing raw_metrics.
CREATE TABLE IF NOT EXISTS qubeley.cpu_metrics
(
    timestamp     DateTime64(3),
    hostname      LowCardinality(String),
    total_percent Float64,
    core_count    UInt16,
    load_1m       Float64,
    load_5m       Float64,
    load_15m      Float64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (hostname, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
