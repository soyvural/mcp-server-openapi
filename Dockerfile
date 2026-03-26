# Multi-stage build for mcp-server-openapi
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty)" \
    -o mcp-server-openapi \
    ./cmd/mcp-server-openapi

# Runtime stage
FROM alpine:3.20

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 mcp && \
    adduser -D -u 1000 -G mcp mcp

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mcp-server-openapi /usr/local/bin/mcp-server-openapi

# Switch to non-root user
USER mcp

# Default command
ENTRYPOINT ["/usr/local/bin/mcp-server-openapi"]
CMD ["--help"]
