# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**USD Exchange Rate WebSocket Monitor** - Real-time USD exchange rate monitoring system with SFTP data collection and WebSocket broadcasting.

## Architecture

### Core Components

1. **SFTP Client** (`main.go`, `validator.go`)
   - Downloads exchange rate data from remote server
   - Validates data before overwriting local files
   - Supports scheduled downloads every second
   - Smart validation to prevent downloading invalid data

2. **Data Processing** (`usd_parser.go`, `json_manager.go`)
   - Parses latest USD rate from `exrate.txt`
   - Converts to JSON format with 2 decimal places
   - Manages market status (open/closed)
   - Supports manual rate updates

3. **WebSocket Server** (`websocket_server.go`)
   - Real-time USD rate broadcasting
   - Automatic updates when files change
   - Manual rate update API
   - Web interface for monitoring

4. **Scheduler** (`scheduler.go`)
   - Downloads data every second from SFTP server
   - Only overwrites local files if server data is valid
   - Updates JSON file automatically

## Key Files Structure

```
aus-websocket/
├── main.go              # Main application entry point
├── usd_parser.go         # USD rate parsing from exrate.txt
├── json_manager.go       # JSON file management with market status
├── websocket_server.go   # WebSocket server and HTTP handlers
├── scheduler.go          # SFTP download scheduler
├── validator.go          # Data validation logic
├── parser.go             # Legacy combined data parser
├── static/
│   └── index.html        # Web interface
├── raw-data/
│   ├── exrate.txt        # Raw USD exchange rate data
│   ├── market_retail.txt # Raw gold market data  
│   └── usd_rate.json     # Processed USD rate JSON
└── .env                  # SFTP configuration

```

## Configuration (.env)

```bash
# SFTP Configuration
SFTP_REMOTE_PATH=/home/webinfo/exrate.txt
SFTP_REMOTE_PATH2=/home/webinfo/market_retail.txt
SFTP_LOCAL_PATH=./raw-data

# Scheduler Configuration  
DOWNLOAD_INTERVAL_SECONDS=1
WEBSOCKET_PORT=8080
```

## Usage Modes

### 1. WebSocket Server (Default)
```bash
./gold-websocket
```
- Starts WebSocket server on port 8080
- Automatically downloads data every second
- Serves web interface at http://localhost:8080

### 2. One-time Download
```bash
./gold-websocket download
```
- Downloads files once and exits

### 3. Continuous Download Only
```bash
./gold-websocket continuous  
```
- Downloads data every second without WebSocket server

## API Endpoints

- `GET /` - Web interface
- `GET /api/data` - Current USD rate JSON
- `POST /api/update-rate` - Manual rate update
- `WebSocket /ws` - Real-time rate updates

### Manual Rate Update
```bash
curl -X POST http://localhost:8080/api/update-rate \
  -H "Content-Type: application/json" \
  -d '{"buy": 32.15, "sell": 32.25}'
```

## Data Flow

1. **SFTP Download**: Downloads `exrate.txt` and `market_retail.txt` every second
2. **Validation**: Checks if server data is valid (non-zero values)
3. **Processing**: Extracts latest USD rate from `exrate.txt`
4. **JSON Creation**: Saves to `usd_rate.json` with market status
5. **Broadcasting**: Sends updates via WebSocket to connected clients

## JSON Data Format

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

## Market Status Logic

- **Open**: Monday-Friday, 9:00-17:00 (excluding 12:00-13:00 lunch)
- **Closed**: Weekends, outside business hours
- **Source**: `auto` (from SFTP) or `manual` (manual update)

## Development Commands

```bash
# Build application
go build -o gold-websocket

# Run with auto SFTP downloads
./gold-websocket

# Test single download
./gold-websocket download

# Run continuous downloads only  
./gold-websocket continuous
```

## Key Features

- **Smart Validation**: Only updates local files if server data is valid
- **Market Status**: Tracks market open/closed status
- **Manual Override**: Allows manual rate updates when market is closed
- **Real-time Updates**: WebSocket broadcasting for instant updates
- **Data Persistence**: JSON file for reliable data storage
- **2 Decimal Precision**: All rates rounded to 2 decimal places

## Error Handling

- Invalid server data is rejected, local files preserved
- Market status automatically determined
- WebSocket reconnection on connection loss
- Fallback from JSON to exrate.txt if needed

## Important Notes

- Always validates server data before overwriting local files
- Supports both automatic (SFTP) and manual rate updates
- Market status affects display but not functionality
- JSON file can be manually edited if needed
- WebSocket broadcasts to all connected clients simultaneously