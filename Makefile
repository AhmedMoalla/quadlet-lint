.DEFAULT_GOAL := build

.PHONY: fmt generate build

fmt:
	go fmt ./...
	go run golang.org/x/tools/cmd/goimports@latest -local 'github.com/AhmedMoalla/quadlet-lint' -w .

generate: fmt
	go generate ./...

build: generate
	go vet ./...
	mkdir -p bin
	go build -o bin ./cmd/quadlet-lint