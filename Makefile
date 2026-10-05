.PHONY: help build run clean test test-race test-coverage lint fmt install setup build-all ps-setup docker-build docker-run check cover

# Default target
help:
	@echo "ShellSage v4.0 - Makefile Commands"
	@echo "======================================"
	@echo ""
	@echo "Available commands:"
	@echo "  make setup         - Setup project (download dependencies & tidy)"
	@echo "  make build         - Build the project binary"
	@echo "  make build-all     - Build cross-platform release binaries"
	@echo "  make run           - Build and run the project interactively"
	@echo "  make test          - Run all unit tests"
	@echo "  make test-race     - Run all unit tests with race detector"
	@echo "  make test-coverage - Run unit tests with code coverage summary"
	@echo "  make lint          - Run go vet and format verification"
	@echo "  make fmt           - Format all code"
	@echo "  make ps-setup      - Verify PowerShell environment module"
	@echo "  make docker-build  - Build minimal Docker container"
	@echo "  make docker-run    - Run ShellSage inside Docker container"
	@echo "  make install       - Install shellsage globally"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make help          - Show this help message"

setup:
	@echo "📦 Setting up project dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Setup complete!"

build:
	@echo "🔨 Building ShellSage v4.0..."
	go build -ldflags="-s -w" -o shellsage ./cmd/shellsage
	@echo "✅ Build complete: ./shellsage"

build-all:
	@echo "🔨 Building cross-platform binaries..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/shellsage-linux-amd64 ./cmd/shellsage
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/shellsage-linux-arm64 ./cmd/shellsage
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/shellsage-darwin-amd64 ./cmd/shellsage
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/shellsage-darwin-arm64 ./cmd/shellsage
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/shellsage-windows-amd64.exe ./cmd/shellsage
	@echo "✅ Cross-platform build complete in dist/"

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

test-coverage:
	@echo "🧪 Coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "✅ Done (view in browser: go tool cover -html=coverage.out)"

cover: test-coverage

lint:
	@echo "🔍 Running linter and format check..."
	go vet ./...
	@unformatted=$$(gofmt -l . | grep -v '^$$' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files require gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "✅ Linting complete!"

fmt:
	@echo "✨ Formatting code..."
	gofmt -l -w .
	@echo "✅ Formatting complete!"

ps-setup:
	@echo "⚙️ Testing PowerShell module script..."
	@if command -v pwsh >/dev/null 2>&1; then \
		pwsh -Command "Import-Module ./scripts/ShellSage.ps1; Test-ShellSageEnvironment"; \
	else \
		echo "PowerShell (pwsh) not found in local environment. Script is located at ./scripts/ShellSage.ps1"; \
	fi

docker-build:
	@echo "🐳 Building Docker image shellsage:latest..."
	docker build -t shellsage:latest .
	@echo "✅ Docker image built!"

docker-run:
	@echo "🐳 Running ShellSage in Docker..."
	docker-compose run --rm shellsage

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf shellsage shellsage.exe dist/ coverage.out
	go clean
	@echo "✅ Clean complete!"

install: build
	@echo "📦 Installing ShellSage to GOPATH/bin..."
	go install ./cmd/shellsage
	@echo "✅ Installation complete!"

check: fmt lint test
	@echo ""
	@echo "✅ All checks passed successfully!"
