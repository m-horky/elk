.DEFAULT_GOAL := check

all: check

.PHONY: check
check:
	golangci-lint fmt --diff
	go test ./...
	go vet ./...
	golangci-lint run

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: build
build:
	mkdir -p build
	go build -o build/elk-identity ./examples/elk-identity
	go build -o build/elk-config ./examples/elk-config
	go build -o build/elk-config-rhsm ./examples/elk-config-rhsm
