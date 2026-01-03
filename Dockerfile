# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /quasar-go ./cmd/quasar

# Final stage
FROM alpine:3.21

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S quasar && adduser -S quasar -G quasar

# Copy binary
COPY --from=builder /quasar-go /usr/local/bin/quasar-go

# Set permissions
RUN chmod +x /usr/local/bin/quasar-go

# Use non-root user
USER quasar

# Default environment
ENV QUASAR_INTERVAL=10

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep quasar-go || exit 1

ENTRYPOINT ["quasar-go"]
