.PHONY: help build run clean test lint fmt install setup

# Default target
help:
	@echo "ShellSage - Makefile Commands"
	@echo "==============================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup    - Setup project (download dependencies)"
	@echo "  make build    - Build the project"
	@echo "  make run      - Build and run the project"
	@echo "  make test     - Run tests"
	@echo "  make lint     - Run linter"
	@echo "  make fmt      - Format code"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make install  - Install shellsage globally"
	@echo "  make help     - Show this help message"

setup:
	@echo "📦 Setting up project..."
	go mod download
	go mod tidy
	@echo "✅ Setup complete!"

build:
	@echo "🔨 Building ShellSage..."
	go build -o shellsage

run: build
	@echo "🚀 Running ShellSage..."
	./shellsage

test:
	@echo "🧪 Running tests..."
	go test -v ./...
	@echo "✅ Tests complete!"

lint:
	@echo "🔍 Running linter..."
	go vet ./...
	@echo "✅ Linting complete!"

fmt:
	@echo "✨ Formatting code..."
	go fmt ./...
	@echo "✅ Formatting complete!"

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f shellsage shellsage.exe
	go clean
	@echo "✅ Clean complete!"

install: build
	@echo "📦 Installing ShellSage..."
	go install
	@echo "✅ Installation complete!"

check: fmt lint test
	@echo "✅ All checks passed!"
