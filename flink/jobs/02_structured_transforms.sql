-- Job 2: Structured transforms
-- Reads from the raw-metrics Kafka topic, parses the JSON payload by
-- metric_type, and routes each type to its structured ClickHouse table.
-- Runs independently from the raw passthrough job.

-- Source: Kafka raw-metrics topic (separate consumer group from job 1)
CREATE TABLE kafka_metrics_source (
    payload     STRING,
    event_time  TIMESTAMP(3) METADATA FROM 'timestamp',
    WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH (
    'connector'                     = 'kafka',
    'topic'                         = 'raw-metrics',
    'properties.bootstrap.servers'  = 'kafka:29092',
    'properties.group.id'           = 'flink-structured-transforms',
    'scan.startup.mode'             = 'earliest-offset',
    'format'                        = 'raw'
);

-- Sink: cpu_metrics
CREATE TABLE clickhouse_cpu_sink (
    timestamp     TIMESTAMP(3),
    hostname      STRING,
    total_percent DOUBLE,
    core_count    INT,
    load_1m       DOUBLE,
    load_5m       DOUBLE,
    load_15m      DOUBLE
) WITH (
    'connector'  = 'jdbc',
    'url'        = 'jdbc:clickhouse://clickhouse:8123/qubeley',
    'table-name' = 'cpu_metrics',
    'username'   = 'qubeley',
    'password'   = 'qubeley_dev',
    'driver'     = 'com.clickhouse.jdbc.ClickHouseDriver'
);

-- Sink: memory_metrics
CREATE TABLE clickhouse_memory_sink (
    timestamp         TIMESTAMP(3),
    hostname          STRING,
    total_bytes       BIGINT,
    available_bytes   BIGINT,
    used_bytes        BIGINT,
    used_percent      DOUBLE,
    swap_total_bytes  BIGINT,
    swap_used_bytes   BIGINT,
    swap_used_percent DOUBLE
) WITH (
    'connector'  = 'jdbc',
    'url'        = 'jdbc:clickhouse://clickhouse:8123/qubeley',
    'table-name' = 'memory_metrics',
    'username'   = 'qubeley',
    'password'   = 'qubeley_dev',
    'driver'     = 'com.clickhouse.jdbc.ClickHouseDriver'
);

-- Sink: temperature_metrics
CREATE TABLE clickhouse_temperature_sink (
    timestamp           TIMESTAMP(3),
    hostname            STRING,
    sensor_key          STRING,
    temperature_celsius DOUBLE,
    high_celsius        DOUBLE,
    critical_celsius    DOUBLE
) WITH (
    'connector'  = 'jdbc',
    'url'        = 'jdbc:clickhouse://clickhouse:8123/qubeley',
    'table-name' = 'temperature_metrics',
    'username'   = 'qubeley',
    'password'   = 'qubeley_dev',
    'driver'     = 'com.clickhouse.jdbc.ClickHouseDriver'
);

-- Sink: logs
CREATE TABLE clickhouse_logs_sink (
    timestamp TIMESTAMP(3),
    hostname  STRING,
    unit      STRING,
    priority  INT,
    message   STRING
) WITH (
    'connector'  = 'jdbc',
    'url'        = 'jdbc:clickhouse://clickhouse:8123/qubeley',
    'table-name' = 'logs',
    'username'   = 'qubeley',
    'password'   = 'qubeley_dev',
    'driver'     = 'com.clickhouse.jdbc.ClickHouseDriver'
);

-- CPU transform
INSERT INTO clickhouse_cpu_sink
SELECT
    event_time,
    JSON_VALUE(payload, '$.hostname')                         AS hostname,
    CAST(JSON_VALUE(payload, '$.total_percent')   AS DOUBLE)  AS total_percent,
    CAST(JSON_VALUE(payload, '$.core_count')      AS INT)     AS core_count,
    CAST(JSON_VALUE(payload, '$.load_average[0]') AS DOUBLE)  AS load_1m,
    CAST(JSON_VALUE(payload, '$.load_average[1]') AS DOUBLE)  AS load_5m,
    CAST(JSON_VALUE(payload, '$.load_average[2]') AS DOUBLE)  AS load_15m
FROM kafka_metrics_source
WHERE JSON_VALUE(payload, '$.metric_type') = 'cpu';

-- Memory transform
INSERT INTO clickhouse_memory_sink
SELECT
    event_time,
    JSON_VALUE(payload, '$.hostname')                              AS hostname,
    CAST(JSON_VALUE(payload, '$.total_bytes')       AS BIGINT)    AS total_bytes,
    CAST(JSON_VALUE(payload, '$.available_bytes')   AS BIGINT)    AS available_bytes,
    CAST(JSON_VALUE(payload, '$.used_bytes')        AS BIGINT)    AS used_bytes,
    CAST(JSON_VALUE(payload, '$.used_percent')      AS DOUBLE)    AS used_percent,
    CAST(JSON_VALUE(payload, '$.swap.total_bytes')  AS BIGINT)    AS swap_total_bytes,
    CAST(JSON_VALUE(payload, '$.swap.used_bytes')   AS BIGINT)    AS swap_used_bytes,
    CAST(JSON_VALUE(payload, '$.swap.used_percent') AS DOUBLE)    AS swap_used_percent
FROM kafka_metrics_source
WHERE JSON_VALUE(payload, '$.metric_type') = 'memory';

-- Temperature transform: flatten readings array into individual rows
INSERT INTO clickhouse_temperature_sink
SELECT
    event_time,
    JSON_VALUE(payload, '$.hostname')                               AS hostname,
    JSON_VALUE(r.reading, '$.sensor_key')                           AS sensor_key,
    CAST(JSON_VALUE(r.reading, '$.temperature_celsius') AS DOUBLE)  AS temperature_celsius,
    CAST(JSON_VALUE(r.reading, '$.high_celsius')        AS DOUBLE)  AS high_celsius,
    CAST(JSON_VALUE(r.reading, '$.critical_celsius')    AS DOUBLE)  AS critical_celsius
FROM kafka_metrics_source
CROSS JOIN UNNEST(
    CAST(JSON_VALUE(payload, '$.readings') AS ARRAY<STRING>)
) AS r (reading)
WHERE JSON_VALUE(payload, '$.metric_type') = 'temperature';

-- Logs transform: flatten entries array into individual rows
INSERT INTO clickhouse_logs_sink
SELECT
    event_time,
    JSON_VALUE(payload, '$.hostname')               AS hostname,
    JSON_VALUE(e.entry, '$.unit')                   AS unit,
    CAST(JSON_VALUE(e.entry, '$.priority') AS INT)  AS priority,
    JSON_VALUE(e.entry, '$.message')                AS message
FROM kafka_metrics_source
CROSS JOIN UNNEST(
    CAST(JSON_VALUE(payload, '$.entries') AS ARRAY<STRING>)
) AS e (entry)
WHERE JSON_VALUE(payload, '$.metric_type') = 'logs';
