# Cosmoflare - Go CLI + library for the Cloudflare developer platform
# Makefile for building, testing, and releasing Cosmoflare
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary info
BINARY_NAME=cosmoflare
BINARY_UNIX=$(BINARY_NAME)_unix
VERSION=$(shell ccs version --short 2>/dev/null | sed 's/ .*//' || grep -o '"version":"[^"]*"' .version-registry.json 2>/dev/null | head -1 | cut -d'"' -f4 || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/CosmoLabs-org/cosmoflare/cmd.AppVersion=$(VERSION) -X github.com/CosmoLabs-org/cosmoflare/cmd.BuildTime=$(BUILD_TIME) -X github.com/CosmoLabs-org/cosmoflare/cmd.GitCommit=$(GIT_COMMIT)"

# Build settings
BUILD_DIR=build
DIST_DIR=dist

# Cross-compilation targets
PLATFORMS=linux/amd64 linux/arm64 linux/armv7 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64
PLATFORMS_MAP=linux_amd64:linux-x86_64 linux_arm64:linux-aarch64 linux_armv7:linux-armv7 windows_amd64:windows-x86_64 windows_arm64:windows-aarch64 darwin_amd64:darwin-x86_64 darwin_arm64:darwin-aarch64

# Docker settings
DOCKER_IMAGE=cosmoflare
DOCKER_TAG=$(VERSION)

# Default target
.PHONY: all
all: clean deps test build

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build for current platform (cosmoflare binary + backward-compat r2go2 alias)
.PHONY: build
build:
	@echo "🏗️  Building $(BINARY_NAME) for $(shell go env GOOS)/$(shell go env GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@ln -sf $(BINARY_NAME) $(BUILD_DIR)/r2go2
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME) (+ $(BUILD_DIR)/r2go2 alias symlink)"

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

# Build specific platform
.PHONY: build-platform
build-platform:
	@if [ -z "$(TARGET)" ]; then \
		echo "Usage: make build-platform TARGET=linux/amd64"; \
		exit 1; \
	fi
	@echo "🏗️  Building $(BINARY_NAME) for $(TARGET)..."
	@mkdir -p $(DIST_DIR)
	@os=$$(echo $(TARGET) | cut -d'/' -f1); \
	arch=$$(echo $(TARGET) | cut -d'/' -f2); \
	output_name=$(BINARY_NAME)-$$os-$$arch; \
	if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
	echo "Building $$os/$$arch -> $$output_name"; \
	CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$$output_name .; \
	echo "✅ Build complete: $(DIST_DIR)/$$output_name"

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

# Create checksums for distribution files
.PHONY: checksums
checksums: dist
	@echo "🔐 Creating checksums..."
	@cd $(DIST_DIR); \
	for file in $(BINARY_NAME)-$(VERSION)-*.{tar.gz,zip}; do \
		if [ -f "$$file" ]; then \
			sha256sum "$$file" >> $(BINARY_NAME)-$(VERSION)-checksums.txt; \
		fi; \
	done
	@echo "✅ Checksums created: $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-checksums.txt"

# Install with Homebrew (local)
.PHONY: install-brew
install-brew: build
	@echo "🍺 Installing with Homebrew..."
	@if [ -d "$(HOME)/.brew" ] || command -v brew >/dev/null; then \
		brew install --cask $(BUILD_DIR)/$(BINARY_NAME); \
	else \
		echo "❌ Homebrew not found. Please install Homebrew first."; \
		exit 1; \
	fi

# Create DEB package
.PHONY: deb
deb: build
	@echo "📦 Creating DEB package..."
	@mkdir -p $(DIST_DIR)/deb/DEBIAN
	@mkdir -p $(DIST_DIR)/deb/usr/local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/deb/usr/local/bin/
	@echo "Package: cosmoflare" > $(DIST_DIR)/deb/DEBIAN/control
	@echo "Version: $(VERSION)" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Section: utils" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Priority: optional" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Architecture: amd64" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Maintainer: CosmoLabs <support@cosmolabs.org>" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Description: Go CLI for the full Cloudflare developer platform" >> $(DIST_DIR)/deb/DEBIAN/control
	@echo "Depends: " >> $(DIST_DIR)/deb/DEBIAN/control
	@dpkg-deb --build $(DIST_DIR)/deb $(DIST_DIR)/cosmoflare_$(VERSION)_amd64.deb
	@rm -rf $(DIST_DIR)/deb
	@echo "✅ DEB package created: $(DIST_DIR)/cosmoflare_$(VERSION)_amd64.deb"

# Create RPM package
.PHONY: rpm
rpm: build
	@echo "📦 Creating RPM package..."
	@mkdir -p $(DIST_DIR)/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/rpmbuild/BUILD/
	@echo "Name: cosmoflare" > $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "Version: $(VERSION)" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "Release: 1%{?dist}" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "Summary: Go CLI for the full Cloudflare developer platform" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "License: MIT" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "URL: https://github.com/CosmoLabs-org/cosmoflare" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "%description" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "Go CLI for the full Cloudflare developer platform (R2, Workers, KV, DNS, D1, and more)." >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "%prep" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "%build" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "%install" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "mkdir -p %{buildroot}/usr/local/bin" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "install -m 755 cosmoflare %{buildroot}/usr/local/bin/" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "%files" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@echo "/usr/local/bin/cosmoflare" >> $(DIST_DIR)/rpmbuild/SPECS/cosmoflare.spec
	@cd $(DIST_DIR)/rpmbuild && rpmbuild -bb SPECS/cosmoflare.spec --define "_topdir $(PWD)"
	@find $(DIST_DIR)/rpmbuild/RPMS -name "*.rpm" -exec cp {} $(DIST_DIR)/ \;
	@rm -rf $(DIST_DIR)/rpmbuild
	@echo "✅ RPM package created in $(DIST_DIR)/"

# Generate SBOM (Software Bill of Materials)
.PHONY: sbom
sbom:
	@echo "📋 Generating SBOM..."
	@if command -v syft >/dev/null 2>&1; then \
		syft . -o cyclonedx-json > $(DIST_DIR)/sbom.cyclonedx.json; \
		syft . -o spdx-json > $(DIST_DIR)/sbom.spdx.json; \
		echo "✅ SBOM created in $(DIST_DIR)/"; \
	else \
		echo "⚠️  Syft not found. Install with: go install github.com/anchore/syft@latest"; \
	fi

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

# Run integration tests (requires CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN)
.PHONY: test-integration
test-integration:
	@echo "Running integration tests against live R2..."
	$(GOTEST) -v -tags=integration -timeout 10m ./pkg/cosmoflare/
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

# Install binary locally (defaults to ~/.local/bin if GOPATH not set)
.PHONY: install
install: build
	@INSTALL_PATH="$${GOPATH:-$$HOME/.local}"/bin; \
	mkdir -p "$$INSTALL_PATH"; \
	echo "📥 Installing $(BINARY_NAME) to $$INSTALL_PATH/cosmoflare..."; \
	cp $(BUILD_DIR)/$(BINARY_NAME) "$$INSTALL_PATH/cosmoflare"; \
	chmod +x "$$INSTALL_PATH/cosmoflare"; \
	echo "✅ Installed successfully!"; \
	if ! echo "$$PATH" | grep -q "$$INSTALL_PATH"; then \
		SHELL_RC="$$HOME/.zshrc"; \
		if [ -f "$$HOME/.bashrc" ] && [ ! -f "$$HOME/.zshrc" ]; then \
			SHELL_RC="$$HOME/.bashrc"; \
		fi; \
		if ! grep -q "$$INSTALL_PATH" "$$SHELL_RC" 2>/dev/null; then \
			echo "" >> "$$SHELL_RC"; \
			echo "# Cosmoflare CLI" >> "$$SHELL_RC"; \
			echo "export PATH=\"$$INSTALL_PATH:\$$PATH\"" >> "$$SHELL_RC"; \
			echo "✅ Added $$INSTALL_PATH to PATH in $$SHELL_RC"; \
		fi; \
		echo ""; \
		echo "🔄 Run this to use cosmoflare now:"; \
		echo "   source $$SHELL_RC"; \
		echo ""; \
		echo "   Or just open a new terminal."; \
	else \
		echo ""; \
		echo "🎉 Ready to use! Run: cosmoflare --help"; \
	fi

# Install to /usr/local/bin (system-wide, prompts for password)
.PHONY: install-system
install-system: build
	@echo "📥 Installing $(BINARY_NAME) to /usr/local/bin/cosmoflare..."
	@echo "🔐 You may be prompted for your password..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/cosmoflare
	@sudo chmod +x /usr/local/bin/cosmoflare
	@sudo ln -sf /usr/local/bin/cosmoflare /usr/local/bin/r2go2
	@echo "✅ Installed successfully! Run: cosmoflare --help (r2go2 alias available)"

# Install case-insensitive aliases (R2Go2, r2go2, R2go2 all work on any OS)
.PHONY: install-aliases
install-aliases: build
	@INSTALL_DIR="$${GOPATH:-$$HOME/.local}"/bin; \
	mkdir -p "$$INSTALL_DIR"; \
	cp $(BUILD_DIR)/$(BINARY_NAME) "$$INSTALL_DIR/$(BINARY_NAME)"; \
	chmod +x "$$INSTALL_DIR/$(BINARY_NAME)"; \
	for name in R2Go2 r2go2 R2go2; do \
		if [ "$$name" != "$(BINARY_NAME)" ]; then \
			ln -sf "$$INSTALL_DIR/$(BINARY_NAME)" "$$INSTALL_DIR/$$name"; \
		fi; \
	done; \
	echo "✅ Installed with aliases: r2go2, R2Go2, R2go2 all work"

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

# ─── Desktop (Tauri) ───────────────────────────────────────────────────────
# The Cosmoflare desktop app (ROAD-063) lives in desktop/. The Go daemon is
# bundled as a Tauri externalBin sidecar, cross-compiled per target triple.
DESKTOP_DIR=desktop

.PHONY: desktop-sidecar
desktop-sidecar: ## Cross-compile the Go daemon as the Tauri sidecar (all triples)
	bash $(DESKTOP_DIR)/scripts/build-sidecar.sh

.PHONY: desktop-dev
desktop-dev: ## Run the desktop app in dev mode (builds the host sidecar first)
	cd $(DESKTOP_DIR) && bun run tauri dev

.PHONY: desktop-build
desktop-build: ## Build the desktop app installers/bundles for the host OS
	cd $(DESKTOP_DIR) && bun run tauri build

.PHONY: desktop-test
desktop-test: ## Run desktop JS (vitest) + Rust (cargo test) suites
	cd $(DESKTOP_DIR) && bun run test
	cd $(DESKTOP_DIR)/src-tauri && cargo test

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
# Security scan (TASK-006): gate releases on known-vulnerability scan.
# Auto-installs govulncheck on first use; fails the release on any finding.
.PHONY: vulncheck
vulncheck:
	@echo "🔎 Running govulncheck (release security gate)..."
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

.PHONY: release-prepare
release-prepare: clean deps test vulncheck build-all dist checksums sbom
	@echo "🚀 Release preparation complete!"
	@echo "📦 Distribution files:"
	@ls -la $(DIST_DIR)/*.{tar.gz,zip,deb,rpm} 2>/dev/null || echo "No distribution files found"
	@echo "🔐 Security files:"
	@ls -la $(DIST_DIR)/*checksums* $(DIST_DIR)/sbom.* 2>/dev/null || echo "No security files found"

# Release (FEAT-016): prepare + sanity-gate + stage + checksums, then PRINT the
# publish command. Publishing stays operator-gated: releases are cut locally,
# no CI (project policy). A platform build that fails mid-way still packs a
# docs-only archive (README/LICENSE, ~4KB, no binary — seen with linux-armv7
# in v0.28.0), so every archive must exceed 1MB before staging.
.PHONY: release
release:
	@if [ -z "$(TAG)" ]; then \
		echo "❌ TAG is required."; \
		echo "Usage: make release TAG=vX.Y.Z"; \
		exit 1; \
	fi
	@$(MAKE) release-prepare
	@echo "🔎 Archive sanity gate (>1MB each, else failed platform build)..."
	@set -e; \
	cd $(DIST_DIR); \
	found=0; \
	for f in $(BINARY_NAME)-$(VERSION)-*.tar.gz $(BINARY_NAME)-$(VERSION)-*.zip; do \
		[ -e "$$f" ] || continue; \
		found=1; \
		size=$$(stat -f %z "$$f" 2>/dev/null || stat -c %s "$$f"); \
		if [ "$$size" -lt 1048576 ]; then \
			echo "❌ Archive $$f is only $$size bytes (<1MB) — failed platform build, refusing to stage"; \
			exit 1; \
		fi; \
		echo "  ✅ $$f ($$size bytes)"; \
	done; \
	if [ "$$found" -eq 0 ]; then \
		echo "❌ No $(BINARY_NAME)-$(VERSION)-*.tar.gz|zip archives found in $(DIST_DIR)/"; \
		exit 1; \
	fi
	@echo "📦 Staging $(DIST_DIR)/upload/..."
	@rm -rf $(DIST_DIR)/upload
	@mkdir -p $(DIST_DIR)/upload
	@cd $(DIST_DIR); \
	for f in $(BINARY_NAME)-$(VERSION)-*.tar.gz $(BINARY_NAME)-$(VERSION)-*.zip; do \
		[ -e "$$f" ] && cp "$$f" upload/ || true; \
	done
	@cd $(DIST_DIR)/upload && shasum -a 256 $(BINARY_NAME)-$(VERSION)-* > checksums-sha256.txt
	@echo "🔐 Checksums written: $(DIST_DIR)/upload/checksums-sha256.txt"
	@echo ""
	@echo "🚀 Release $(TAG) is staged and ready. Publish it yourself with:"
	@echo ""
	@echo "  gh release create $(TAG) dist/upload/* --repo CosmoLabs-org/cosmoflare --title $(TAG) --notes-file $$(ls docs/release-notes/$(BINARY_NAME)-$(TAG)-ReleaseNotes*.md 2>/dev/null | head -1 || echo "docs/release-notes/$(BINARY_NAME)-$(TAG)-ReleaseNotes.md")"
	@echo ""
	@echo "⚠️  Not run automatically — publishing is operator-gated by policy (local releases, no CI)."

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
	@ccs version --bump patch

# Increment version (minor)
.PHONY: version-minor
version-minor:
	@echo "🔢 Incrementing minor version..."
	@ccs version --bump minor

# Increment version (major)
.PHONY: version-major
version-major:
	@echo "🔢 Incrementing major version..."
	@ccs version --bump major

# Show help
.PHONY: help
help:
	@echo "📚 Cosmoflare Makefile Commands"
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
	@echo "  install-aliases Install with case-insensitive symlinks (r2go2, R2Go2, R2go2)"
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
	@echo "  release        Stage a release and print the publish command: make release TAG=vX.Y.Z"
	@echo "  checksums     Create SHA256 checksums for release files"
	@echo "  deb           Create DEB package (Ubuntu/Debian)"
	@echo "  rpm           Create RPM package (RHEL/Fedora)"
	@echo "  sbom          Generate Software Bill of Materials"
	@echo "  deps          Install/update dependencies"
	@echo "  run           Build and run the application"
	@echo ""
	@echo "Package Management:"
	@echo "  build-platform Build for specific platform: TARGET=linux/amd64"
	@echo "  install-brew  Install locally via Homebrew"
	@echo ""
	@echo "Other:"
	@echo "  help          Show this help message"
	@echo "  tidy          Tidy go modules"
	@echo "  check         Quick quality check (fmt + vet + test)"
	@echo "  url           Show dev URL (via portless)"

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

# Tidy go modules
.PHONY: tidy
tidy:
	@echo "📦 Tidying modules..."
	$(GOMOD) tidy

# Quick quality check (fmt + vet + test)
.PHONY: check
check: fmt vet test
	@echo "✅ All checks passed"

# Show dev URL (via portless)
.PHONY: url
url:
	@if command -v ccs >/dev/null 2>&1; then \
		ccs url 2>/dev/null || echo "⚠️  Portless not configured"; \
	else \
		echo "⚠️  ccs not found"; \
	fi