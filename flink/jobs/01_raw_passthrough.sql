-- Job 1: Raw passthrough
-- Reads every message from the raw-metrics Kafka topic and writes the full
-- JSON payload verbatim to qubeley.raw_metrics in ClickHouse.
-- This job must be running before the structured transform job.

-- Source: Kafka raw-metrics topic
CREATE TABLE kafka_raw_source (
    `timestamp`   TIMESTAMP(3) METADATA FROM 'timestamp',
    payload       STRING
) WITH (
    'connector'                     = 'kafka',
    'topic'                         = 'raw-metrics',
    'properties.bootstrap.servers'  = 'kafka:29092',
    'properties.group.id'           = 'flink-raw-passthrough',
    'scan.startup.mode'             = 'earliest-offset',
    'format'                        = 'raw'
);

-- Sink: ClickHouse raw_metrics table
CREATE TABLE clickhouse_raw_sink (
    `timestamp`   TIMESTAMP(3),
    hostname      STRING,
    metric_type   STRING,
    payload       STRING
) WITH (
    'connector'  = 'jdbc',
    'url'        = 'jdbc:clickhouse://clickhouse:8123/qubeley',
    'table-name' = 'raw_metrics',
    'username'   = 'qubeley',
    'password'   = 'qubeley_dev',
    'driver'     = 'com.clickhouse.jdbc.ClickHouseDriver'
);

-- Extract hostname and metric_type from the JSON payload so ClickHouse
-- can partition and filter efficiently, while still storing the full blob.
INSERT INTO clickhouse_raw_sink
SELECT
    `timestamp`,
    JSON_VALUE(payload, '$.hostname')    AS hostname,
    JSON_VALUE(payload, '$.metric_type') AS metric_type,
    payload
FROM kafka_raw_source;