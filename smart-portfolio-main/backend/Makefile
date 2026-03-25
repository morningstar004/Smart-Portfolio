# =============================================================================
# Smart Portfolio — Makefile
# =============================================================================
# Usage:
#   make              — build the binary
#   make run          — build and run locally
#   make dev          — run with live reload (requires air)
#   make test         — run all tests
#   make test-v       — run all tests with verbose output
#   make cover        — run tests with coverage report
#   make lint         — run go vet + staticcheck
#   make fmt          — format all Go files
#   make tidy         — tidy and verify go modules
#   make docker-up    — start docker compose stack
#   make docker-down  — stop docker compose stack
#   make docker-build — build the Docker image
#   make migrate      — run migrations against local DB
#   make clean        — remove build artifacts
#   make help         — show this help
# =============================================================================

.PHONY: all build run dev test test-v cover lint fmt tidy \
        docker-up docker-down docker-build docker-logs \
        migrate clean help check

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

APP_NAME     := smart-portfolio
BINARY       := bin/$(APP_NAME)
CMD_DIR      := ./cmd/server
MIGRATIONS   := ./migrations

GO           := go
GOFLAGS      := -trimpath
LDFLAGS      := -s -w

# Docker
COMPOSE      := docker compose
IMAGE_NAME   := smart-portfolio
IMAGE_TAG    := latest

# Database (local development defaults — override via environment)
DATABASE_URL ?= postgres://portfolio:portfolio_secret@localhost:5432/smart_portfolio?sslmode=disable

# ---------------------------------------------------------------------------
# Default target
# ---------------------------------------------------------------------------

all: build

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

## build: Compile the server binary into ./bin/
build:
	@echo "==> Building $(BINARY)..."
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(CMD_DIR)
	@echo "==> Done: $(BINARY)"

## build-linux: Cross-compile a static Linux binary (for containers / CI)
build-linux:
	@echo "==> Cross-compiling for Linux amd64..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build \
		$(GOFLAGS) -ldflags="$(LDFLAGS) -extldflags '-static'" \
		-o $(BINARY)-linux-amd64 $(CMD_DIR)
	@echo "==> Done: $(BINARY)-linux-amd64"

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------

## run: Build and run the server locally
run: build
	@echo "==> Starting $(APP_NAME)..."
	@./$(BINARY)

## dev: Run with live reload using air (install: go install github.com/air-verse/air@latest)
dev:
	@command -v air >/dev/null 2>&1 || { echo "Error: 'air' not found. Install with: go install github.com/air-verse/air@latest"; exit 1; }
	air

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

## test: Run all tests
test:
	@echo "==> Running tests..."
	$(GO) test ./... -count=1

## test-v: Run all tests with verbose output
test-v:
	@echo "==> Running tests (verbose)..."
	$(GO) test ./... -v -count=1

## cover: Run tests with coverage and open the HTML report
cover:
	@echo "==> Running tests with coverage..."
	@mkdir -p coverage
	$(GO) test ./... -count=1 -coverprofile=coverage/coverage.out -covermode=atomic
	$(GO) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "==> Coverage report: coverage/coverage.html"

## bench: Run benchmarks
bench:
	@echo "==> Running benchmarks..."
	$(GO) test ./... -bench=. -benchmem -run=^$$ -count=1

# ---------------------------------------------------------------------------
# Code quality
# ---------------------------------------------------------------------------

## lint: Run go vet and staticcheck
lint: vet
	@echo "==> Running staticcheck..."
	@command -v staticcheck >/dev/null 2>&1 || { echo "Installing staticcheck..."; go install honnef.co/go/tools/cmd/staticcheck@latest; }
	staticcheck ./...
	@echo "==> Lint passed."

## vet: Run go vet
vet:
	@echo "==> Running go vet..."
	$(GO) vet ./...

## fmt: Format all Go source files
fmt:
	@echo "==> Formatting Go files..."
	$(GO) fmt ./...
	@echo "==> Done."

## tidy: Tidy and verify module dependencies
tidy:
	@echo "==> Tidying modules..."
	$(GO) mod tidy
	$(GO) mod verify
	@echo "==> Modules tidy."

## check: Run fmt, vet, lint, and tests — the full pre-commit suite
check: fmt vet lint test
	@echo "==> All checks passed."

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

## docker-build: Build the Docker image
docker-build:
	@echo "==> Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "==> Docker image built."

## docker-up: Start the docker compose stack (app)
docker-up:
	@echo "==> Starting docker compose stack..."
	$(COMPOSE) up -d --build
	@echo "==> Stack is running. App: http://localhost:8080/healthz"

## docker-down: Stop the docker compose stack
docker-down:
	@echo "==> Stopping docker compose stack..."
	$(COMPOSE) down

## docker-logs: Tail logs from the docker compose stack
docker-logs:
	$(COMPOSE) logs -f

## docker-logs-app: Tail only the app container logs
docker-logs-app:
	$(COMPOSE) logs -f app

## docker-ps: Show running containers
docker-ps:
	$(COMPOSE) ps

# ---------------------------------------------------------------------------
# Database / Migrations
# ---------------------------------------------------------------------------

## migrate: Run SQL migrations against DATABASE_URL (local dev)
migrate:
	@echo "==> Running migrations against $(DATABASE_URL)..."
	@for f in $(MIGRATIONS)/*.sql; do \
		echo "  -> Applying $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f" || exit 1; \
	done
	@echo "==> Migrations complete."

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

## clean: Remove build artifacts, coverage, and temp files
clean:
	@echo "==> Cleaning up..."
	rm -rf bin/ coverage/ tmp/
	$(GO) clean -cache -testcache
	@echo "==> Clean."

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------

## help: Show this help message
help:
	@echo ""
	@echo "Smart Portfolio — Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | \
		sed 's/^## //' | \
		column -t -s ':' | \
		sed 's/^/  /'
	@echo ""
