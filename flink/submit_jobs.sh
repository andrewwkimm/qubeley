#!/usr/bin/env bash
# submit_jobs.sh submits both Flink SQL jobs to the running jobmanager.
# Run this after the stack is healthy: make up && bash flink/submit_jobs.sh

set -euo pipefail

JOBMANAGER="qubeley-flink-jobmanager"

echo "Submitting job 1: raw passthrough..."
docker exec -i "$JOBMANAGER" \
    /opt/flink/bin/sql-client.sh \
    --file /opt/flink/jobs/01_raw_passthrough.sql

echo "Submitting job 2: structured transforms..."
docker exec -i "$JOBMANAGER" \
    /opt/flink/bin/sql-client.sh \
    --file /opt/flink/jobs/02_structured_transforms.sql

echo "Both jobs submitted. Check status at http://localhost:8081"