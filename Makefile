# R2Go2 - Cloudflare R2 CLI Tool
# Makefile for building, testing, and releasing R2Go2
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary info
BINARY_NAME=r2go2
BINARY_UNIX=$(BINARY_NAME)_unix
VERSION=$(shell cat VERSION)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion=$(VERSION) -X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.BuildTime=$(BUILD_TIME) -X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.GitCommit=$(GIT_COMMIT)"

# Build settings
BUILD_DIR=build
DIST_DIR=dist

# Cross-compilation targets
PLATFORMS=linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# Docker settings
DOCKER_IMAGE=r2go2
DOCKER_TAG=$(VERSION)

# Default target
.PHONY: all
all: clean deps test build

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build for current platform
.PHONY: build
build:
	@echo "🏗️  Building $(BINARY_NAME) for $(shell go env GOOS)/$(shell go env GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "🏗️  Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building $$os/$$arch -> $$output_name"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$$output_name .; \
	done
	@echo "✅ All builds complete in $(DIST_DIR)/"

# Create distribution archives
.PHONY: dist
dist: build-all
	@echo "📦 Creating distribution archives..."
	@cd $(DIST_DIR); \
	for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		if [ $$os = "windows" ]; then \
			zip -r $(BINARY_NAME)-$(VERSION)-$$os-$$arch.zip $$output_name ../README.md ../LICENSE; \
		else \
			tar -czf $(BINARY_NAME)-$(VERSION)-$$os-$$arch.tar.gz $$output_name ../README.md ../LICENSE; \
		fi; \
	done
	@echo "✅ Distribution archives created in $(DIST_DIR)/"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "📊 Coverage report generated: $(BUILD_DIR)/coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "⚡ Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Run race condition tests
.PHONY: test-race
test-race:
	@echo "🏃 Running race condition tests..."
	$(GOTEST) -race ./...

# Lint code
.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "💅 Formatting code..."
	$(GOCMD) fmt ./...

# Vet code
.PHONY: vet
vet:
	@echo "🔎 Vetting code..."
	$(GOCMD) vet ./...

# Install binary locally
.PHONY: install
install: build
	@echo "📥 Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Install to /usr/local/bin (requires sudo)
.PHONY: install-system
install-system: build
	@echo "📥 Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Development build with race detection
.PHONY: dev
dev:
	@echo "🔧 Building development version with race detection..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run the application
.PHONY: run
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

# Docker run
.PHONY: docker-run
docker-run:
	@echo "🐳 Running Docker container..."
	docker run --rm -it $(DOCKER_IMAGE):latest

# Release preparation
.PHONY: release-prepare
release-prepare: clean deps test build-all dist
	@echo "🚀 Release preparation complete!"
	@echo "📦 Distribution files:"
	@ls -la $(DIST_DIR)/*.{tar.gz,zip} 2>/dev/null || echo "No distribution files found"

# Version management
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Increment version (patch)
.PHONY: version-patch
version-patch:
	@echo "🔢 Incrementing patch version..."
	$(eval NEW_VERSION=$(shell bump2version patch --dry-run --list | grep new_version= | cut -d= -f2))
	@bump2version patch
	@echo "Version updated to $(NEW_VERSION)"

# Increment version (minor)
.PHONY: version-minor
version-minor:
	@echo "🔢 Incrementing minor version..."
	$(eval NEW_VERSION=$(shell bump2version minor --dry-run --list | grep new_version= | cut -d= -f2))
	@bump2version minor
	@echo "Version updated to $(NEW_VERSION)"

# Increment version (major)
.PHONY: version-major
version-major:
	@echo "🔢 Incrementing major version..."
	$(eval NEW_VERSION=$(shell bump2version major --dry-run --list | grep new_version= | cut -d= -f2))
	@bump2version major
	@echo "Version updated to $(NEW_VERSION)"

# Show help
.PHONY: help
help:
	@echo "📚 R2Go2 Makefile Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  build         Build binary for current platform"
	@echo "  build-all     Build binary for all platforms"
	@echo "  dist          Create distribution archives"
	@echo "  dev           Build development version with race detection"
	@echo "  clean         Clean build artifacts"
	@echo ""
	@echo "Testing Commands:"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  bench         Run benchmarks"
	@echo "  test-race     Run race condition tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           Format code"
	@echo "  vet           Vet code"
	@echo "  lint          Run linter (requires golangci-lint)"
	@echo ""
	@echo "Installation:"
	@echo "  install       Install to GOPATH/bin"
	@echo "  install-system Install to /usr/local/bin (requires sudo)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-run    Run Docker container"
	@echo ""
	@echo "Version Management:"
	@echo "  version       Show current version info"
	@echo "  version-patch Increment patch version"
	@echo "  version-minor Increment minor version"
	@echo "  version-major Increment major version"
	@echo ""
	@echo "Release:"
	@echo "  release-prepare Prepare full release with builds and dists"
	@echo "  deps          Install/update dependencies"
	@echo "  run           Build and run the application"
	@echo ""
	@echo "Other:"
	@echo "  help          Show this help message"

# Check dependencies
.PHONY: check-deps
check-deps:
	@echo "🔍 Checking dependencies..."
	@which go > /dev/null || (echo "❌ Go is not installed" && exit 1)
	@which git > /dev/null || (echo "❌ Git is not installed" && exit 1)
	@echo "✅ All required dependencies are installed"

# Integration tests (requires real R2 credentials)
.PHONY: test-integration
test-integration:
	@echo "🧪 Running integration tests..."
	@if [ -z "$(CLOUDFLARE_API_TOKEN)" ] || [ -z "$(CLOUDFLARE_ACCOUNT_ID)" ]; then \
		echo "❌ CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID must be set for integration tests"; \
		exit 1; \
	fi
	@echo "⚠️  Integration tests will create real R2 resources. Proceed with caution."
	@read -p "Continue? (y/N) " confirm && [ "$$confirm" = "y" ] || exit 1
	$(GOTEST) -v -tags=integration ./...