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

################################################################################

.PHONY: \
	build \
	clean \
	help \
	lint \
	reformat \
	test \
	type_check