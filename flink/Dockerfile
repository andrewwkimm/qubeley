FROM flink:1.18.0-scala_2.12

ARG FLINK_VERSION=1.18.0
ARG KAFKA_CONNECTOR_VERSION=3.1.0-1.18
ARG FLINK_JDBC_VERSION=3.1.2-1.18
ARG CLICKHOUSE_JDBC_VERSION=0.4.6

RUN curl -fSL \
    "https://repo1.maven.org/maven2/org/apache/flink/flink-sql-connector-kafka/${KAFKA_CONNECTOR_VERSION}/flink-sql-connector-kafka-${KAFKA_CONNECTOR_VERSION}.jar" \
    -o /opt/flink/lib/flink-sql-connector-kafka.jar && \
    curl -fSL \
    "https://repo1.maven.org/maven2/org/apache/flink/flink-connector-jdbc/${FLINK_JDBC_VERSION}/flink-connector-jdbc-${FLINK_JDBC_VERSION}.jar" \
    -o /opt/flink/lib/flink-connector-jdbc.jar && \
    curl -fSL \
    "https://repo1.maven.org/maven2/com/clickhouse/clickhouse-jdbc/${CLICKHOUSE_JDBC_VERSION}/clickhouse-jdbc-${CLICKHOUSE_JDBC_VERSION}-all.jar" \
    -o /opt/flink/lib/clickhouse-jdbc.jar