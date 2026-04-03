# Ausiris Bullion Price Feed

Real-time precious metals price feed service. Fetches price data via SFTP, parses market data, and broadcasts updates via WebSocket and Redis pub/sub.

## Architecture

This repository is one of three in the Ausiris Bullion system:

| Component | Repository | Path |
|-----------|-----------|------|
| Web Frontend | [ausiris-bullion-web](../ausiris-bullion-web) | `/Users/teerayutht/WorkSpace/ausiris-bullion-web` |
| API Server | [ausiris-bullion-api](../ausiris-bullion-api) | `/Users/teerayutht/WorkSpace/ausiris-bullion-api` |
| Price Feed (this) | [ausiris-bullion-price-feed](../ausiris-bullion-price-feed) | `/Users/teerayutht/WorkSpace/ausiris-bullion-price-feed` |

## Tech Stack

- **Language**: Go 1.24.4
- **WebSocket**: github.com/coder/websocket
- **Pub/Sub**: Redis (github.com/redis/go-redis/v9)
- **SFTP**: github.com/pkg/sftp
- **Config**: github.com/joho/godotenv
- **Deployment**: Docker Compose (dev, local, uat, prod)

## Getting Started

### Prerequisites

- Go 1.24.4+
- Redis running locally or accessible
- SFTP credentials for the price data source

### Installation

```bash
# Install Go dependencies
go mod download
```

### Configuration

Create `.env` with your Redis connection, SFTP credentials, and scheduler settings.

### Building

```bash
# Using Makefile
make build

# Or directly with Go
go build -ldflags="-w -s" -o gold-socket ./cmd/server
```

### Running

```bash
# Run the binary
./gold-socket
```

### Docker

```bash
# Development environment
docker compose -f docker-compose.dev.yml up

# Local environment
docker compose -f docker-compose.local.yml up
```

## Project Structure

```
ausiris-bullion-price-feed/
├── cmd/
│   └── server/
│       └── main.go         # Entry point
├── internal/
│   ├── api/               # HTTP/WebSocket API handlers and routes
│   ├── config/            # Configuration management
│   ├── logger/            # Logging utilities
│   ├── parser/            # Price data parsing (exchange rates, market data)
│   ├── redis/             # Redis client and pub/sub
│   ├── scheduler/         # Scheduled price fetch tasks
│   ├── sftp/              # SFTP client for data source connection
│   └── websocket/         # WebSocket hub and client management
├── pkg/
│   └── models/            # Shared data models (market_data, usd_rate)
├── deployments/           # Deployment configurations
├── docker-compose.*.yml   # Environment-specific compose files
├── Makefile
├── go.mod
├── go.sum
├── gold-socket            # Compiled binary (Go 1.24.4 module name)
└── gold-websocket         # Legacy compiled binary
```

## Data Flow

1. **SFTP Fetch**: Scheduled connector retrieves price files from the data source
2. **Parse**: Raw files are parsed into structured market data (gold prices, FX rates)
3. **Redis Pub/Sub**: Prices are published to Redis for the API server to consume
4. **WebSocket**: Real-time WebSocket server broadcasts prices to connected frontend clients

## Available Makefile Commands

```bash
make build          # Build for current platform
make build-linux    # Cross-compile for Linux AMD64
make build-darwin   # Build for macOS ARM64
make run            # Build and run
make docker         # Build Docker image
make docker-run     # Run with Docker Compose
make clean          # Remove build artifacts
make fmt            # Format Go source code
make lint           # Run golangci-lint
```

## Cross-Repo Dependencies

- **ausiris-bullion-web**: Frontend connects to this service's WebSocket endpoint for real-time price updates
- **ausiris-bullion-api**: API server subscribes to Redis pub/sub channels for price data

## License

Private - Ausiris Bullion Internal Use Only
