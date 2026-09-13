# Multi-stage build for R2Go2 CLI
# Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)

# Build stage — toolchain MUST match go.mod's go directive (BUG-050: 1.25
# vs go 1.26 caused silent toolchain downloads / build failures).
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set build environment
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-w -s \
    -X github.com/CosmoLabs-org/cosmoflare/cmd.AppVersion=${VERSION} \
    -X github.com/CosmoLabs-org/cosmoflare/cmd.BuildTime=${BUILD_TIME} \
    -X github.com/CosmoLabs-org/cosmoflare/cmd.GitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo -o cosmoflare .

# Final stage — pinned minor release, never :latest (BUG-050)
FROM alpine:3.21

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S r2go2 && \
    adduser -u 1001 -S r2go2 -G r2go2

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/cosmoflare .

# Create directories for configuration
RUN mkdir -p /app/config && \
    chown -R r2go2:r2go2 /app

# Switch to non-root user
USER r2go2

# Expose configuration directory
VOLUME ["/app/config"]

# Environment variables
ENV R2GO2_CONFIG_DIR=/app/config

# Set entrypoint
ENTRYPOINT ["/app/cosmoflare"]

# Default command
CMD ["--help"]

# Labels
LABEL maintainer="CosmoLabs <support@cosmolabs.org>" \
      org.opencontainers.image.title="Cosmoflare" \
      org.opencontainers.image.description="Go CLI and library for the full Cloudflare developer platform" \
      org.opencontainers.image.url="https://github.com/CosmoLabs-org/cosmoflare" \
      org.opencontainers.image.source="https://github.com/CosmoLabs-org/cosmoflare" \
      org.opencontainers.image.vendor="CosmoLabs" \
      org.opencontainers.image.licenses="MIT"