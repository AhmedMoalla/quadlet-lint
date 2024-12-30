.DEFAULT_GOAL := build

.PHONY: build lint test

build:
	go generate -v ./...
	mkdir -p bin
	go build -v -o bin ./cmd/quadlet-lint

# TODO Add coverage reporting
test:
	go test ./...

lint:
	golangci-lint run