-- Raw metrics stores the unprocessed JSON payload from Kafka verbatim.
-- This serves as an audit log and replay source if Flink jobs need to be
-- re-run against historical data after Kafka's retention window expires.
CREATE TABLE IF NOT EXISTS qubeley.raw_metrics
(
    timestamp    DateTime64(3),
    hostname     LowCardinality(String),
    metric_type  LowCardinality(String),
    payload      String
)
ENGINE = MergeTree
PARTITION BY (toYYYYMM(timestamp), metric_type)
ORDER BY (hostname, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
