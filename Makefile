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

test:
	go test -v -race ./...

type_check:
	staticcheck ./...

################################################################################

clean:
	go clean -cache
	go clean -fuzzcache

run:
	go run cmd/qubeley/main.go

up:
	docker compose up -d

down:
	docker compose down

flink-submit:
	docker exec -i qubeley-flink-jobmanager \
		/opt/flink/bin/sql-client.sh \
		--file /opt/flink/jobs/01_raw_passthrough.sql
	docker exec -i qubeley-flink-jobmanager \
		/opt/flink/bin/sql-client.sh \
		--file /opt/flink/jobs/02_structured_transforms.sql

reset-clickhouse:
	docker compose down
	docker volume rm qubeley_clickhouse-data
	docker compose up -d

################################################################################

.PHONY: \
	build \
	clean \
	down \
	flink-submit \
	help \
	lint \
	reformat \
	reset-clickhouse \
	run \
	test \
	type_check \
	up