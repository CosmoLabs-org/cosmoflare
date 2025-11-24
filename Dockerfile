# Multi-stage build for R2Go2 CLI
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)

# Build stage
FROM golang:1.25-alpine AS builder

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
    -X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion=${VERSION} \
    -X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.BuildTime=${BUILD_TIME} \
    -X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.GitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo -o r2go2 .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S r2go2 && \
    adduser -u 1001 -S r2go2 -G r2go2

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/r2go2 .

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
ENTRYPOINT ["/app/r2go2"]

# Default command
CMD ["--help"]

# Labels
LABEL maintainer="CosmoLabs <support@cosmolabs.org>" \
      org.opencontainers.image.title="R2Go2" \
      org.opencontainers.image.description="Cloudflare R2 CLI management tool" \
      org.opencontainers.image.url="https://github.com/CosmoLabs-org/CosmoDev-R2Go2" \
      org.opencontainers.image.source="https://github.com/CosmoLabs-org/CosmoDev-R2Go2" \
      org.opencontainers.image.vendor="CosmoLabs" \
      org.opencontainers.image.licenses="MIT"