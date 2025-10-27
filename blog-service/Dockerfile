# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /blog-service ./cmd

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /blog-service .

EXPOSE 8083

CMD ["./blog-service"]
