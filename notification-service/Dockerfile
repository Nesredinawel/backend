# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service ./cmd


# --- Runtime stage ---
FROM alpine:latest

WORKDIR /root/

# Create keys directory inside container
RUN mkdir -p /keys

# Copy Firebase credentials from the host machine
# NOTE: this path assumes Dockerfile is inside notification-service/
COPY ../keys/firebase-service-account.json /keys/firebase-service-account.json

# Copy the compiled binary from the builder
COPY --from=builder /notification-service .

# Expose the notification service port
EXPOSE 8084

# Run the service
CMD ["./notification-service"]
