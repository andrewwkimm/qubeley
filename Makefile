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

consume:
	go run cmd/consumer/main.go cmd/consumer/clickhouse.go

run:
	go run cmd/qubeley/main.go

up:
	docker compose up -d

down:
	docker compose down

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
	reformat \
	reset-clickhouse \
	run \
	test \
	type_check \
	up