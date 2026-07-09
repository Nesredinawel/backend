FROM golang:1.24-alpine AS builder

WORKDIR /app

# Build all services in parallel
COPY api-gateway/go.mod api-gateway/go.sum ./gateway/
WORKDIR /app/gateway
RUN go mod download
COPY api-gateway/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api-gateway ./cmd

WORKDIR /app/auth
COPY auth-service/go.mod auth-service/go.sum ./
RUN go mod download
COPY auth-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/auth-service ./cmd

WORKDIR /app/mood
COPY mood-service/go.mod mood-service/go.sum ./
RUN go mod download
COPY mood-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/mood-service ./cmd

WORKDIR /app/blog
COPY blog-service/go.mod blog-service/go.sum ./
RUN go mod download
COPY blog-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/blog-service ./cmd

WORKDIR /app/notification
COPY notification-service/go.mod notification-service/go.sum ./
RUN go mod download
COPY notification-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd

WORKDIR /app/docs
COPY api-docs-service/go.mod api-docs-service/go.sum ./
RUN go mod download
COPY api-docs-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api-docs-service ./cmd

# =========================
# Runtime
# =========================
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/* /usr/local/bin/
COPY api-docs-service/docs /root/docs

EXPOSE 8081

COPY deploy/start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
