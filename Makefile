# GitHub Stars Notify - Makefile
# This Makefile provides standardized commands for development and CI/CD

# Variables
BINARY_NAME = github-stars-notify
DOCKER_LINT_IMAGE = golangci/golangci-lint:v1.61.0
COVERAGE_FILE = coverage.out

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOFMT = gofmt

# Default target
.PHONY: all
all: quality-checks build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  help           - Show this help message"
	@echo "  all            - Run quality checks and build"
	@echo "  quality-checks - Run all quality checks (fmt, lint, test)"
	@echo "  fmt            - Format Go code"
	@echo "  fmt-check      - Check Go code formatting"
	@echo "  lint           - Run golangci-lint using Docker"
	@echo "  test           - Run tests"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  build          - Build the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy go.mod and go.sum"
	@echo "  run            - Run the application locally"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"

# Quality checks target (mirrors GitHub Actions)
.PHONY: quality-checks
quality-checks: fmt-check lint test-race
	@echo "✅ All quality checks passed!"

# Format Go code
.PHONY: fmt
fmt:
	@echo "📝 Formatting Go code..."
	$(GOFMT) -s -w .
	@echo "✅ Code formatted"

# Check Go code formatting
.PHONY: fmt-check
fmt-check:
	@echo "📝 Checking Go code formatting..."
	@if [ "$$($(GOFMT) -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ The following files are not formatted:"; \
		$(GOFMT) -s -l .; \
		echo "Run 'make fmt' to format your code."; \
		exit 1; \
	fi
	@echo "✅ Go formatting check passed"

# Lint using Docker
.PHONY: lint
lint:
	@echo "🔍 Running golangci-lint using Docker..."
	@if command -v docker >/dev/null 2>&1; then \
		docker run --rm \
			-v $(PWD):/app \
			-w /app \
			$(DOCKER_LINT_IMAGE) \
			golangci-lint run --timeout=5m; \
		echo "✅ Linting passed"; \
	else \
		echo "❌ Docker not found. Please install Docker to run linting."; \
		exit 1; \
	fi

# Download dependencies
.PHONY: deps
deps:
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	@echo "✅ Dependencies downloaded"

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "🧹 Tidying go.mod and go.sum..."
	$(GOMOD) tidy
	@echo "✅ Dependencies tidied"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...
	@echo "✅ Tests passed"

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "🧪 Running tests with race detection..."
	$(GOTEST) -v -race ./...
	@echo "✅ Tests with race detection passed"

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./...
	@if [ -f $(COVERAGE_FILE) ]; then \
		echo "📊 Test coverage:"; \
		$(GOCMD) tool cover -func=$(COVERAGE_FILE) | tail -1; \
	fi
	@echo "✅ Tests with coverage completed"

# Build the application
.PHONY: build
build: deps
	@echo "🏗️  Building $(BINARY_NAME)..."
	$(GOBUILD) -v -o $(BINARY_NAME) .
	@echo "✅ Build successful"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	@echo "✅ Clean completed"

# Run the application locally
.PHONY: run
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(BINARY_NAME) .
	@echo "✅ Docker image built"

# Run Docker container
.PHONY: docker-run
docker-run: docker-build
	@echo "🐳 Running Docker container..."
	docker run --rm -it $(BINARY_NAME)

# Development targets
.PHONY: dev-setup
dev-setup: deps tidy
	@echo "🔧 Setting up development environment..."
	@echo "✅ Development environment ready"

# Pre-commit checks (what developers should run before committing)
.PHONY: pre-commit
pre-commit: quality-checks
	@echo "🎉 Pre-commit checks passed! Ready to commit."

# CI target (what GitHub Actions runs)
.PHONY: ci
ci: quality-checks build
	@echo "🎯 CI checks completed successfully" 