.DEFAULT_GOAL := build

.PHONY: fmt generate build

fmt:
	go fmt ./...
	go run golang.org/x/tools/cmd/goimports@latest -local 'github.com/AhmedMoalla/quadlet-lint' -w .

generate: fmt
	go generate -v ./...

build: generate
	go vet ./...
	mkdir -p bin
	go build -v -o bin ./cmd/quadlet-lint