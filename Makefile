.PHONY: help build run test test-unit test-integration coverage lint fmt vet \
        docker-up docker-down docker-build migrate mock clean

# Application
APP_NAME := search-engine-service
BUILD_DIR := bin
MAIN_PATH := ./cmd/api

# Go
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

# Docker
DOCKER_COMPOSE := docker-compose

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

# ============================================================================
# DEVELOPMENT
# ============================================================================

## run: Run the application locally
run:
	$(GO) run $(MAIN_PATH)/main.go

## build: Build the application binary
build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	$(GO) clean -cache

# ============================================================================
# TESTING
# ============================================================================

## test: Run all tests
test:
	$(GO) test ./... -v

## test-unit: Run unit tests only (exclude integration)
test-unit:
	$(GO) test ./... -v -short

## test-integration: Run integration tests only
test-integration:
	$(GO) test ./... -v -run Integration

## coverage: Run tests with coverage report
coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## coverage-func: Show coverage by function
coverage-func:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

# ============================================================================
# CODE QUALITY
# ============================================================================

## lint: Run golangci-lint
lint:
	golangci-lint run -v

## fmt: Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	$(GO) vet ./...

## check: Run all code quality checks
check: fmt vet lint test

# ============================================================================
# DOCKER
# ============================================================================

## docker-up: Start all services with docker-compose
docker-up:
	$(DOCKER_COMPOSE) up -d

## docker-down: Stop all services
docker-down:
	$(DOCKER_COMPOSE) down

## docker-build: Build docker images
docker-build:
	$(DOCKER_COMPOSE) build

## docker-logs: Show docker logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-ps: Show running containers
docker-ps:
	$(DOCKER_COMPOSE) ps

# ============================================================================
# MOCK SERVERS
# ============================================================================

## mock: Run mock provider servers locally
mock:
	@echo "Starting mock servers..."
	@cd mock/provider_a && $(GO) run main.go &
	@cd mock/provider_b && $(GO) run main.go &
	@echo "Provider A: http://localhost:8081/api/contents"
	@echo "Provider B: http://localhost:8082/feed"

## mock-stop: Stop mock servers
mock-stop:
	@echo "Stopping mock servers..."
	@lsof -ti:8081 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8082 | xargs kill -9 2>/dev/null || true
	@echo "Mock servers stopped"

# ============================================================================
# DATABASE
# ============================================================================

## migrate: Run database migrations
migrate:
	$(GO) run $(MAIN_PATH)/main.go migrate

## migrate-down: Rollback last migration
migrate-down:
	$(GO) run $(MAIN_PATH)/main.go migrate down

# ============================================================================
# DEPENDENCIES
# ============================================================================

## deps: Download dependencies
deps:
	$(GO) mod download

## deps-tidy: Tidy dependencies
deps-tidy:
	$(GO) mod tidy

## deps-verify: Verify dependencies
deps-verify:
	$(GO) mod verify

# ============================================================================
# TOOLS
# ============================================================================

## tools: Install development tools
tools:
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest


## swagger: Serve swagger UI (requires api/openapi.yaml)
swagger:
	@echo "Open http://localhost:8090 in your browser"
	npx @redocly/cli preview-docs api/openapi.yaml --port 8090
