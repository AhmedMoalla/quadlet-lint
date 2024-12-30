.DEFAULT_GOAL := build

.PHONY: generate build lint test

generate:
	go generate -v ./...

build: generate
	mkdir -p bin
	go build -v -o bin ./cmd/quadlet-lint

# TODO Add coverage reporting
test:
	go test ./...

lint:
	golangci-lint run