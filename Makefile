.DEFAULT_GOAL := build
GOBIN ?= $$(go env GOPATH)/bin

.PHONY: generate build lint test install-go-test-coverage test-coverage

generate:
	go generate -v ./...

build: generate
	mkdir -p bin
	go build -v -o bin ./cmd/quadlet-lint

install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

test: generate
	go test ./... -v -race -coverprofile=./cover.out -covermode=atomic -coverpkg=./...

test-coverage: test install-go-test-coverage
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

lint:
	golangci-lint run