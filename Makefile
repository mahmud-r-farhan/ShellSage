.PHONY: help build run clean test test-race lint fmt install setup docker-build docker-run

# Default target
help:
	@echo "ShellSage v3.0 - Makefile Commands"
	@echo "======================================"
	@echo ""
	@echo "Available commands:"
	@echo "  make setup        - Setup project (download dependencies & tidy)"
	@echo "  make build        - Build the project binary"
	@echo "  make run          - Build and run the project interactively"
	@echo "  make test         - Run all unit tests"
	@echo "  make test-race    - Run all unit tests with race detector"
	@echo "  make lint         - Run go vet"
	@echo "  make fmt          - Format all code"
	@echo "  make docker-build - Build minimal Docker container"
	@echo "  make docker-run   - Run ShellSage inside Docker container"
	@echo "  make install      - Install shellsage globally"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make help         - Show this help message"

setup:
	@echo "📦 Setting up project dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Setup complete!"

build:
	@echo "🔨 Building ShellSage v3.0..."
	go build -ldflags="-s -w" -o shellsage .
	@echo "✅ Build complete: ./shellsage"

run: build
	@echo "🚀 Running ShellSage..."
	./shellsage

test:
	@echo "🧪 Running unit tests..."
	go test -v ./...
	@echo "✅ Tests complete!"

test-race:
	@echo "🧪 Running tests with race detector..."
	go test -v -race ./...
	@echo "✅ Race tests complete!"

lint:
	@echo "🔍 Running linter..."
	go vet ./...
	@echo "✅ Linting complete!"

fmt:
	@echo "✨ Formatting code..."
	go fmt ./...
	@echo "✅ Formatting complete!"

docker-build:
	@echo "🐳 Building Docker image shellsage:latest..."
	docker build -t shellsage:latest .
	@echo "✅ Docker image built!"

docker-run:
	@echo "🐳 Running ShellSage in Docker..."
	docker-compose run --rm shellsage

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f shellsage shellsage.exe dist/*
	go clean
	@echo "✅ Clean complete!"

install: build
	@echo "📦 Installing ShellSage to GOPATH/bin..."
	go install
	@echo "✅ Installation complete!"

check: fmt lint test
	@echo "✅ All checks passed successfully!"
