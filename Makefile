APP_NAME=jobqueue
BIN_DIR=bin
MAIN_PATH=./cmd/server

# Build variables
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

.PHONY: help build run test clean tidy fmt lint docker-build docker-run

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)

run: ## Run the application
	go run $(MAIN_PATH)

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

tidy: ## Tidy Go modules
	go mod tidy
	go mod verify

fmt: ## Format Go code
	go fmt ./...

lint: ## Lint Go code
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

dev: ## Run in development mode with hot reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		go run $(MAIN_PATH); \
	fi

docker-build: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) .

docker-run: ## Run Docker container
	docker run -p 8080:8080 $(APP_NAME):$(VERSION)

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest

setup: tidy install-tools ## Setup development environment
	@echo "Development environment setup complete!"
