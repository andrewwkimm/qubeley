# Qubeley

**Qubeley** is a real-time systems metrics pipeline, powered by Kafka, Clickhouse, and Grafana.

## Overview

- **Collector** (`cmd/qubeley`) runs on the host and gathers CPU, memory, temperature, and system log metrics every 5 seconds, publishing each as a JSON message to Kafka
- **Consumer** (`cmd/consumer`) reads from Kafka and writes structured rows into ClickHouse
- **Grafana** queries ClickHouse directly and displays a live system metrics dashboard

Below is how the final product will look like:

<img width="2556" height="1352" alt="grafana" src="https://github.com/user-attachments/assets/52fa82d5-9daf-42fe-917f-c5c1a640e9f3" />

## Requirements

- [Go](https://go.dev/dl/) 1.24+
- [Docker](https://docs.docker.com/get-docker/) with the Compose plugin

## Getting started

**1. Clone the repo**

```bash
git clone https://github.com/andrewwkimm/qubeley.git
cd qubeley
```

**2. Start the infrastructure**

```bash
make up
```

Wait for Kafka and ClickHouse to become healthy:

```bash
docker compose ps
```

Both should show `healthy` before proceeding. This typically takes 30–60 seconds.

**3. Run the ClickHouse migrations**

The migrations run automatically on first start via `docker-entrypoint-initdb.d`. If the tables are missing (e.g. the volume pre-existed), run them manually:

```bash
make migrations
```

Verify:

```bash
docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --query "SHOW TABLES FROM qubeley"
```

You should see: `cpu_metrics`, `logs`, `memory_metrics`, `raw_metrics`, `temperature_metrics`.

**4. Start the collector and consumer**

In two separate terminals:

```bash
# Terminal 1 — collects metrics and publishes to Kafka
make run

# Terminal 2 — reads from Kafka and writes to ClickHouse
make consume
```

**5. Open Grafana**

Navigate to [http://localhost:3000](http://localhost:3000) and log in with `admin / admin`.

The **System Metrics** dashboard is auto-provisioned. Set the time range to **Last 5 minutes** to see live data.

**6. Stopping**

```bash
# Ctrl+C in both terminal windows, then:
make down
```

Data is preserved in Docker volumes. `make up` will resume from where you left off.

## Contributing

To contribute to this project, run the setup target after cloning the repository.

```bash
make setup
```

To test changes locally, run:

```bash
make build
```