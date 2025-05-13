# ===== STAGE 1: Build Stage =====
FROM golang:1.24 AS builder

# Set environment variables
ENV CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

# Install necessary build dependencies
RUN apt-get update && apt-get install -y gcc libc-dev

# Set working directory inside the container
WORKDIR /app

# Copy Go modules manifests first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go binary
RUN go build -o /app/server ./cmd/main.go

# ===== STAGE 2: Runtime Stage =====
FROM debian:stable-slim

# Install required runtime dependencies (for CGO)
RUN apt-get update && apt-get install -y ca-certificates libc6 && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the compiled binary from builder stage
COPY --from=builder /app/server .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./server"]
