-- Log entries written by Flink after processing raw_metrics.
-- Priority follows syslog conventions: 0=Emergency, 3=Error, 6=Info, 7=Debug.
CREATE TABLE IF NOT EXISTS qubeley.logs
(
    timestamp DateTime64(3),
    hostname  LowCardinality(String),
    unit      LowCardinality(String),
    priority  UInt8,
    message   String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (hostname, unit, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
