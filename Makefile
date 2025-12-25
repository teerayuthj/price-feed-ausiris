# Gold Socket - Makefile
# Real-time Exchange Rate WebSocket Monitor

# Variables
BINARY_NAME=gold-socket
GO=go
GOFLAGS=-ldflags="-w -s"
DOCKER_COMPOSE=docker compose

# Default target
.PHONY: all
all: build

# ============================================================================
# Build targets
# ============================================================================

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/server

.PHONY: build-linux
build-linux:
	@echo "Building for Linux AMD64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/server

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS ARM64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/server

.PHONY: build-all
build-all: build-linux build-darwin
	@echo "Built all platforms"

# ============================================================================
# Run targets
# ============================================================================

.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GO) run ./cmd/server

.PHONY: run-download
run-download:
	@echo "Running one-time download..."
	$(GO) run ./cmd/server download

.PHONY: run-continuous
run-continuous:
	@echo "Running continuous downloads..."
	$(GO) run ./cmd/server continuous

# ============================================================================
# Development targets
# ============================================================================

.PHONY: local-dev
local-dev:
	@echo "Starting local development environment..."
	@echo "Services: App (hot reload) + Redis"
	@echo "App will be available at http://localhost:8080"
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml up --build

.PHONY: local-dev-detach
local-dev-detach:
	@echo "Starting local development environment (detached)..."
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml up --build -d

.PHONY: local-dev-down
local-dev-down:
	@echo "Stopping development environment..."
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml down

.PHONY: local-dev-logs
local-dev-logs:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml logs -f

.PHONY: local-dev-restart
local-dev-restart:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml restart

# ============================================================================
# Docker targets (Production)
# ============================================================================

.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

.PHONY: docker-up
docker-up:
	@echo "Starting production containers..."
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping production containers..."
	$(DOCKER_COMPOSE) down

.PHONY: docker-logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-restart
docker-restart:
	$(DOCKER_COMPOSE) restart

.PHONY: docker-clean
docker-clean:
	@echo "Cleaning up Docker resources..."
	$(DOCKER_COMPOSE) down -v --rmi local

.PHONY: docker-ps
docker-ps:
	$(DOCKER_COMPOSE) ps

# ============================================================================
# Test targets
# ============================================================================

.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	$(GO) test -race -v ./...

# ============================================================================
# Lint and format
# ============================================================================

.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -l -w . || true

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# ============================================================================
# Dependencies
# ============================================================================

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: tidy
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy

# ============================================================================
# Development tools
# ============================================================================

.PHONY: tools
tools:
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/air-verse/air@latest

# ============================================================================
# Clean
# ============================================================================

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	$(GO) clean -cache

# ============================================================================
# Help
# ============================================================================

.PHONY: help
help:
	@echo "Gold Socket - Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build          Build the application"
	@echo "  build-linux    Build for Linux AMD64"
	@echo "  build-darwin   Build for macOS ARM64"
	@echo "  build-all      Build for all platforms"
	@echo ""
	@echo "Run:"
	@echo "  run            Run the application locally"
	@echo "  run-download   Run one-time SFTP download"
	@echo "  run-continuous Run continuous downloads only"
	@echo ""
	@echo "Development:"
	@echo "  local-dev      Start dev environment (Docker Compose)"
	@echo "  local-dev-down Stop dev environment"
	@echo "  local-dev-logs View dev environment logs"
	@echo ""
	@echo "Docker (Production):"
	@echo "  docker-build   Build Docker images"
	@echo "  docker-up      Start production containers"
	@echo "  docker-down    Stop production containers"
	@echo "  docker-logs    View container logs"
	@echo "  docker-clean   Clean up Docker resources"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-race      Run tests with race detector"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint           Run linter"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps           Download dependencies"
	@echo "  deps-update    Update dependencies"
	@echo "  tidy           Tidy go modules"
	@echo ""
	@echo "Other:"
	@echo "  tools          Install development tools"
	@echo "  clean          Clean build artifacts"
	@echo "  help           Show this help message"
