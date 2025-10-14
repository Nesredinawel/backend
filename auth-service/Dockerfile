# --- Build stage ---
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first (to cache dependencies)
COPY go.mod go.sum ./

# Install git for private repos or modules
RUN apk add --no-cache git

# Download Go dependencies
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the Go binary (statically linked)
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd

# --- Runtime stage ---
FROM alpine:3.18

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /auth-service /auth-service

# Set environment variables
ENV PORT=8081

# Expose the port the app runs on
EXPOSE 8081

# Run the binary
ENTRYPOINT ["/auth-service"]
