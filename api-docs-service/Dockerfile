# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-docs-service ./cmd

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /api-docs-service ./

# Mount docs folder
VOLUME ["/root/docs"]

EXPOSE 8085

CMD ["./api-docs-service"]
