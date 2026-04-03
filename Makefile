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
# UAT Deployment
# ============================================================================

.PHONY: deploy-uat
deploy-uat:
	@echo "🧪 Deploying UAT environment..."
	docker compose -f docker-compose.uat.yml up --build -d
	@echo "✅ UAT deployed!"
	@echo "   WebSocket: http://localhost:8081"
	@echo "   Health:    http://localhost:8081/health"

.PHONY: down-uat
down-uat:
	@echo "Stopping UAT environment..."
	docker compose -f docker-compose.uat.yml down

.PHONY: restart-uat
restart-uat: down-uat deploy-uat

.PHONY: logs-uat
logs-uat:
	docker compose -f docker-compose.uat.yml logs -f

.PHONY: status-uat
status-uat:
	@echo "UAT container status:"
	docker compose -f docker-compose.uat.yml ps

.PHONY: health-uat
health-uat:
	@echo "Checking UAT health..."
	curl -s http://localhost:8081/health && echo "" || echo "(UAT not running)"

.PHONY: clean-uat
clean-uat:
	@echo "Cleaning up UAT resources..."
	docker compose -f docker-compose.uat.yml down -v --rmi all
	@echo "✅ UAT cleanup complete!"

# ============================================================================
# Prod Deployment
# ============================================================================

.PHONY: deploy-prod
deploy-prod:
	@echo "🚀 Deploying PRODUCTION environment..."
	docker compose -f docker-compose.prod.yml up --build -d
	@echo "✅ Production deployed!"
	@echo "   WebSocket: http://localhost:8080"
	@echo "   Health:    http://localhost:8080/health"

.PHONY: down-prod
down-prod:
	@echo "Stopping PRODUCTION environment..."
	docker compose -f docker-compose.prod.yml down

.PHONY: restart-prod
restart-prod: down-prod deploy-prod

.PHONY: logs-prod
logs-prod:
	docker compose -f docker-compose.prod.yml logs -f

.PHONY: status-prod
status-prod:
	@echo "Production container status:"
	docker compose -f docker-compose.prod.yml ps

.PHONY: health-prod
health-prod:
	@echo "Checking Production health..."
	curl -s http://localhost:8080/health && echo "" || echo "(Production not running)"

.PHONY: clean-prod
clean-prod:
	@echo "Cleaning up Production resources..."
	docker compose -f docker-compose.prod.yml down -v --rmi all
	@echo "✅ Production cleanup complete!"

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
	@echo "  deploy-prod      Deploy production (Docker Compose)"
	@echo "  down-prod        Stop production containers"
	@echo "  restart-prod     Restart production"
	@echo "  logs-prod        View production logs"
	@echo "  status-prod      Show production status"
	@echo "  health-prod      Health check production"
	@echo "  clean-prod       Clean production resources"
	@echo ""
	@echo "UAT Environment:"
	@echo "  deploy-uat       Deploy UAT (Docker Compose)"
	@echo "  down-uat         Stop UAT containers"
	@echo "  restart-uat      Restart UAT"
	@echo "  logs-uat         View UAT logs"
	@echo "  status-uat       Show UAT status"
	@echo "  health-uat       Health check UAT"
	@echo "  clean-uat        Clean UAT resources"
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
