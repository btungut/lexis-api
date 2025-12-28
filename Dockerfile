# Build stage
FROM golang:1.25-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to strip debug info and reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o lexis-api \
    .

# Final stage - minimal runtime image
FROM alpine:3.19

# Metadata labels
LABEL maintainer="Burak Tungut"
LABEL license="Lexis Non-Commercial Source License (NCSL) v1.0"
LABEL copyright="Copyright (c) 2025 Burak Tungut"
LABEL commercial-use="prohibited"
LABEL contact="burak.tungut@tungops.com.tr"

# Install CA certificates for HTTPS requests (if needed)
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appuser /build/lexis-api .

# Switch to non-root user
USER appuser

# Run the application
CMD ["./lexis-api"]
