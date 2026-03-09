help:
	cat Makefile

################################################################################

build:
	go mod download
	make reformat
	make lint
	make type_check
	go build ./...
	make test

lint:
	go vet ./...
	golangci-lint run --fix ./...

reformat:
	go fmt ./...

setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	go install honnef.co/go/tools/cmd/staticcheck@v0.5.1
	go install golang.org/x/tools/cmd/godoc@v0.29.0

test:
	go test -v -race ./...

type_check:
	staticcheck ./...

################################################################################

clean:
	go clean -cache
	go clean -fuzzcache

consume:
	go run cmd/consumer/main.go cmd/consumer/clickhouse.go

run:
	go run cmd/qubeley/main.go

up:
	docker compose up -d

down:
	docker compose down

migrations:
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev < clickhouse/migrations/01_create_database.sql
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --database qubeley < clickhouse/migrations/02_create_raw_metrics.sql
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --database qubeley < clickhouse/migrations/03_create_cpu_metrics.sql
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --database qubeley < clickhouse/migrations/04_create_memory_metrics.sql
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --database qubeley < clickhouse/migrations/05_create_temperature_metrics.sql
	docker exec -i qubeley-clickhouse clickhouse-client --user qubeley --password qubeley_dev --database qubeley < clickhouse/migrations/06_create_logs.sql

reset-clickhouse:
	docker compose down
	docker volume rm qubeley_clickhouse-data
	docker compose up -d

################################################################################

.PHONY: \
	build \
	clean \
	consume \
	down \
	help \
	lint \
	migrations \
	reformat \
	reset-clickhouse \
	run \
	test \
	type_check \
	up