# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget tzdata

# Set timezone to Asia/Bangkok
ENV TZ=Asia/Bangkok
ENV WEBSOCKET_PORT=8080

WORKDIR /app

# Copy binary and static files
COPY --from=builder /app/server .
COPY web/static ./web/static

# Create data directory
RUN mkdir -p /app/raw-data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD sh -c "wget -q --spider http://localhost:${WEBSOCKET_PORT}/health || exit 1"

CMD ["./server"]
