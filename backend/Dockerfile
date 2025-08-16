# syntax=docker/dockerfile:1.7
# Multi-stage build for optimized Go backend with security hardening
ARG GO_VERSION=1.21
ARG ALPINE_VERSION=3.19

# Builder stage with build dependencies and cache optimization
FROM docker.io/golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# Install security updates and build dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata \
    && update-ca-certificates

# Create non-root user for build process
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy go mod files and download dependencies (separate layer for caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build the application with optimizations and security flags
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o main ./cmd

# Production stage with minimal footprint
FROM scratch AS production

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd file for non-root user
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /app/main /main

# Use non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/main", "-health-check"] || exit 1

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/main"]

# Development stage for debugging
FROM builder AS development

# Install debugging tools
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy source code
COPY . .

# Build with debug symbols
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -gcflags="all=-N -l" \
    -o main ./cmd

EXPOSE 8080 40000

CMD ["dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./main"]