.DEFAULT_GOAL := build

.PHONY: fmt vet build run help

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -o bin/app ./cmd/server/main.go

run: build
	./bin/app

help:
	@echo "Available targets:"
	@echo "  fmt     - Format Go source code"
	@echo "  vet     - Run go vet on all packages"
	@echo "  build   - Build the application binary"
	@echo "  run     - Build and run the application"
	@echo "  help    - Show this help message"
