
# ==========================================
# Stage 1: Build Go application
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install certificates and Git if dependencies require it
RUN apk add --no-cache ca-certificates git

# Copy dependency files first for Docker layer caching
COPY go.mod go.sum ./

RUN go mod download

# Copy application source
COPY . .

# Build the RLS server
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath \
    -ldflags="-s -w" \
    -o /rate-limiter ./cmd/server

# ==========================================
# Stage 2: Runtime image
# ==========================================
FROM alpine:3.23

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

# Copy compiled binary
COPY --from=builder /rate-limiter /app/rate-limiter

# Run as non-root user
USER appuser

# Go RLS gRPC port
EXPOSE 8081

# Default configuration
ENV RLS_ADDR=:8081
ENV REDIS_ADDR=redis:6379
ENV RATE_LIMIT_DOMAIN=backend-api

CMD ["/app/rate-limiter"]
