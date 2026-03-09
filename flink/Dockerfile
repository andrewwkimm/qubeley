FROM flink:1.18.0-scala_2.12

# Connector versions pinned for reproducibility.
# Kafka connector must match the Flink version exactly.
# ClickHouse connector is maintained separately by ClickHouse Inc.
ARG FLINK_VERSION=1.18.0
ARG KAFKA_CONNECTOR_VERSION=3.1.0-1.18
ARG CLICKHOUSE_CONNECTOR_VERSION=0.6.0

RUN curl -fSL \
    "https://repo1.maven.org/maven2/org/apache/flink/flink-sql-connector-kafka/${KAFKA_CONNECTOR_VERSION}/flink-sql-connector-kafka-${KAFKA_CONNECTOR_VERSION}.jar" \
    -o /opt/flink/lib/flink-sql-connector-kafka.jar && \
    curl -fSL \
    "https://repo1.maven.org/maven2/com/clickhouse/flink/flink-connector-clickhouse/${CLICKHOUSE_CONNECTOR_VERSION}/flink-connector-clickhouse-${CLICKHOUSE_CONNECTOR_VERSION}.jar" \
    -o /opt/flink/lib/flink-connector-clickhouse.jar