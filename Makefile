# Build variables
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
BINARY_NAME := quasar-go
DOCKER_IMAGE := carllee/quasar-go-agent

# Directories
BUILD_DIR := ./dist
CMD_DIR := ./cmd/quasar

.PHONY: all build clean test tidy run dev docker help

## Default target
all: build

## Build the binary
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY_NAME)"

## Build for all platforms
build-all:
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "✅ Built binaries for all platforms"

## Build debug tool
build-debug-tool:
	@echo "🔨 Building debug-process..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/debug-process ./cmd/debug_process
	@echo "✅ Built: $(BUILD_DIR)/debug-process"

## Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

## Tidy dependencies
tidy:
	@echo "📦 Tidying dependencies..."
	$(GOMOD) tidy

## Run locally (development)
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	QUASAR_SERVICE=dev-test $(BUILD_DIR)/$(BINARY_NAME)

## Run with hot reload (requires entr)
dev:
	@echo "🔄 Starting development mode..."
	find . -name "*.go" | entr -r make run

## Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .
	@echo "✅ Built: $(DOCKER_IMAGE):latest"

## Push Docker image
docker-push:
	@echo "🚀 Pushing Docker image..."
	docker push $(DOCKER_IMAGE):latest
	@echo "✅ Pushed: $(DOCKER_IMAGE):latest"

## Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "✅ Cleaned"

## Show help
help:
	@echo "Quasar Agent - Makefile targets:"
	@echo ""
	@echo "  make build       - Build binary for current platform"
	@echo "  make build-all   - Build binaries for all platforms"
	@echo "  make test        - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make tidy        - Tidy go modules"
	@echo "  make run         - Build and run locally"
	@echo "  make dev         - Run with hot reload"
	@echo "  make docker      - Build Docker image"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make help        - Show this help"
