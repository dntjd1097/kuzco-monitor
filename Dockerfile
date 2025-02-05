# 빌드 스테이지
FROM golang:1.23.4-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o kuzco-monitor .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/kuzco-monitor .

# Create config directory
RUN mkdir -p /app/config

# Run as non-root user
RUN adduser -D -g '' appuser
RUN chown -R appuser:appuser /app
USER appuser

# Command to run
CMD ["./kuzco-monitor"] 