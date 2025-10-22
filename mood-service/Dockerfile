# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /mood-service ./cmd

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /root/

# Copy the compiled binary from builder
COPY --from=builder /mood-service .

# Expose mood service port
EXPOSE 8082

# Run the service
CMD ["./mood-service"]
