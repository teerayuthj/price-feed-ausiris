# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**Gold Socket** - Real-time USD exchange rate and gold price monitoring system with SFTP data collection, WebSocket broadcasting, and Redis support for scaling.

## Architecture

### Project Structure

```
gold-socket/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration loading (.env, environment)
│   ├── sftp/
│   │   ├── client.go              # SFTP connection and download logic
│   │   └── validator.go           # File validation logic
│   ├── parser/
│   │   ├── exrate.go              # USD rate parsing from exrate.txt
│   │   ├── market.go              # Market retail data parsing
│   │   └── types.go               # Shared types and utilities
│   ├── websocket/
│   │   ├── hub.go                 # WebSocket hub (client management)
│   │   ├── client.go              # Individual client handling
│   │   └── handler.go             # HTTP upgrade handler
│   ├── scheduler/
│   │   └── scheduler.go           # Periodic download scheduler
│   ├── redis/
│   │   ├── client.go              # Redis connection and operations
│   │   ├── cache.go               # Rate caching
│   │   └── pubsub.go              # Pub/Sub for WebSocket scaling
│   └── api/
│       ├── handler.go             # REST API handlers
│       └── routes.go              # Route registration
├── pkg/
│   └── models/
│       ├── usd_rate.go            # USDRate, USDRateWithStatus structs
│       └── market_data.go         # MarketData, SpotData, GoldData structs
├── web/
│   └── static/
│       └── index.html             # Frontend HTML
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile             # Multi-stage production build
│   │   └── Dockerfile.dev         # Development with hot reload
│   ├── nginx/
│   │   └── nginx.conf             # Nginx reverse proxy config
│   └── systemd/
│       └── gold-socket.service    # Systemd service file
├── raw-data/                       # Data directory (gitignored)
│   ├── exrate.txt
│   ├── market_retail.txt
│   ├── usd_rate.json
│   └── market_data.json
├── docker-compose.yml              # Production Docker Compose
├── docker-compose.dev.yml          # Development Docker Compose
├── Makefile                        # Build automation
├── .air.toml                       # Hot reload config
├── .env.example                    # Environment template
├── go.mod                          # Module: gold-socket
└── go.sum
```

### Core Components

1. **WebSocket Server** (`internal/websocket/`)
   - Uses `github.com/coder/websocket` (modern, context-based API)
   - Hub pattern for managing client connections
   - Real-time data broadcasting

2. **SFTP Client** (`internal/sftp/`)
   - Downloads exchange rate data from remote server
   - Smart validation before overwriting local files

3. **Data Parsers** (`internal/parser/`)
   - Parses latest USD rate from `exrate.txt`
   - Processes market retail data from `market_retail.txt`
   - Market status (open/closed) logic

4. **Redis Integration** (`internal/redis/`)
   - Pub/Sub for multi-instance WebSocket scaling
   - Rate caching with 5-minute TTL
   - Optional - works without Redis

5. **Scheduler** (`internal/scheduler/`)
   - Downloads data every N seconds (configurable)
   - Triggers WebSocket broadcasts

## Configuration (.env)

```bash
# SFTP Configuration
SFTP_HOST=192.168.30.2
SFTP_PORT=22
SFTP_USER=goldsp
SFTP_PASSWORD=your_password
SFTP_REMOTE_PATH=/home/webinfo/exrate.txt
SFTP_REMOTE_PATH2=/home/webinfo/market_retail.txt
SFTP_LOCAL_PATH=./raw-data

# Server Configuration
WEBSOCKET_PORT=8080
STATIC_DIR=./web/static
DATA_DIR=./raw-data

# Scheduler Configuration
DOWNLOAD_INTERVAL_SECONDS=1

# Redis Configuration (Optional)
REDIS_ENABLED=false
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Development Commands

```bash
# Build
make build              # Build binary
make build-linux        # Build for Linux
make build-darwin       # Build for macOS

# Run locally
make run                # Run application
make run-download       # One-time SFTP download
make run-continuous     # Continuous downloads only

# Development with Docker
make local-dev          # Start dev environment (App + Redis)
make local-dev-down     # Stop dev environment
make local-dev-logs     # View logs

# Production Docker
make docker-build       # Build images
make docker-up          # Start production containers
make docker-down        # Stop containers
make docker-logs        # View logs

# Testing & Quality
make test               # Run tests
make test-coverage      # Run with coverage
make lint               # Run linter
make fmt                # Format code

# Dependencies
make deps               # Download dependencies
make deps-update        # Update dependencies
make tidy               # Tidy modules
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Web interface |
| `/ws` | WebSocket | Real-time data streaming |
| `/health` | GET | Health check |
| `/api/data` | GET | Current USD rate JSON |
| `/api/market-data` | GET | Current market data JSON |
| `/api/update-rate` | POST | Manual rate update |

### Manual Rate Update

```bash
curl -X POST http://localhost:8080/api/update-rate \
  -H "Content-Type: application/json" \
  -d '{"buy": 32.15, "sell": 32.25}'
```

## Data Flow

```
SFTP Server (exrate.txt, market_retail.txt)
           ↓
    Scheduler (every N seconds)
           ↓
    SFTP Client → Validator
           ↓
    Local Files (raw-data/)
           ↓
    Parsers → JSON Files
           ↓
    WebSocket Hub → Broadcast
           ↓
    Connected Clients + Redis Pub/Sub
```

## JSON Data Formats

### USD Rate (`usd_rate.json`)
```json
{
  "timestamp": "2025-07-24T15:30:00Z",
  "time": "15:30:00",
  "currency": "USD",
  "buy": 32.15,
  "sell": 32.25,
  "market_status": "open",
  "source": "auto"
}
```

### Market Data (`market_data.json`)
```json
{
  "timestamp": "2025-07-24T15:30:00Z",
  "time": "15:30:00",
  "spot_usd": {"bid": 33.65, "offer": 33.66},
  "g965b_retail": {"bid": 51325.00, "offer": 51375.00},
  "g9999kg_retail": {"bid": 3489592.00, "offer": 3492872.00},
  "g9999g": {"bid": 52885.00, "offer": 53245.00},
  "market_status": "open",
  "source": "auto"
}
```

## Market Status Logic

- **Open**: Monday-Friday, 9:00-17:00 (excluding 12:00-13:00 lunch)
- **Closed**: Weekends, outside business hours
- **Source**: `auto` (from SFTP) or `manual` (manual update)

## Deployment

### Docker Compose (Production)

```bash
# Start all services (App + Nginx + Redis)
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

### Systemd

```bash
# Copy service file
sudo cp deployments/systemd/gold-socket.service /etc/systemd/system/

# Create user
sudo useradd -r -s /sbin/nologin gold-socket

# Install binary
sudo mkdir -p /opt/gold-socket
sudo cp gold-socket /opt/gold-socket/
sudo cp -r web/static /opt/gold-socket/web/
sudo cp .env /opt/gold-socket/
sudo chown -R gold-socket:gold-socket /opt/gold-socket

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable gold-socket
sudo systemctl start gold-socket
```

## Key Libraries

| Package | Purpose |
|---------|---------|
| `github.com/coder/websocket` | WebSocket server (context-based) |
| `github.com/pkg/sftp` | SFTP client |
| `github.com/redis/go-redis/v9` | Redis client |
| `github.com/fsnotify/fsnotify` | File watching |
| `github.com/joho/godotenv` | Environment loading |

## Key Features

- **Modern WebSocket**: Uses `coder/websocket` with context-based operations
- **Redis Scaling**: Pub/Sub for multi-instance support
- **Rate Caching**: Redis caching with 5-minute TTL
- **Smart Validation**: Only updates if server data is valid
- **Docker Ready**: Production and development Docker Compose
- **Hot Reload**: Air for development hot reload
- **Graceful Shutdown**: Context-based cancellation
