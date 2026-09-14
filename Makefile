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
