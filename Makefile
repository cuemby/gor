# Gor Framework Makefile
# Rails-inspired web framework for Go

# Variables
BINARY_NAME := gor
CMD_DIR := ./cmd/gor
PKG_DIR := ./pkg/...
INTERNAL_DIR := ./internal/...
EXAMPLES_DIR := ./examples
BUILD_DIR := ./build
BIN_DIR := ./bin
COVERAGE_FILE := coverage_output/coverage.out
COVERAGE_HTML := coverage_output/coverage.html

# Go commands
GO := go
GOCMD := $(GO)
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Colors for terminal output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# OS Detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	OS := linux
endif
ifeq ($(UNAME_S),Darwin)
	OS := darwin
endif
ifeq ($(UNAME_S),Windows_NT)
	OS := windows
	BINARY_NAME := $(BINARY_NAME).exe
endif

ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
endif
ifeq ($(UNAME_M),arm64)
	ARCH := arm64
endif

# Default target
.DEFAULT_GOAL := help

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build
build: ## Build the gor CLI binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BIN_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)Build complete: $(BIN_DIR)/$(BINARY_NAME)$(NC)"

.PHONY: build-all
build-all: build ## Build CLI and all example applications
	@echo "$(GREEN)Building all examples...$(NC)"
	@for dir in $(shell ls -d $(EXAMPLES_DIR)/*/); do \
		if [ -f $$dir/main.go ]; then \
			echo "Building $$dir..."; \
			$(GOBUILD) -o $$dir/$$(basename $$dir) $$dir/main.go; \
		fi \
	done
	@echo "$(GREEN)All builds complete!$(NC)"

.PHONY: install
install: build ## Install gor binary to GOPATH/bin
	@echo "$(GREEN)Installing $(BINARY_NAME) to $(GOPATH)/bin...$(NC)"
	@cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "$(GREEN)Installation complete!$(NC)"

.PHONY: clean
clean: ## Clean build artifacts and temporary files
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR) $(BIN_DIR)
	@rm -rf coverage_output
	@rm -f $(EXAMPLES_DIR)/*/*.test
	@rm -f $(EXAMPLES_DIR)/*/$(shell basename $(EXAMPLES_DIR)/*)
	@find . -name "*.db" -type f -delete 2>/dev/null || true
	@find . -name "*.db-journal" -type f -delete 2>/dev/null || true
	@echo "$(GREEN)Clean complete!$(NC)"

# =============================================================================
# Testing Targets
# =============================================================================

.PHONY: test
test: ## Run all tests
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "$(GREEN)Running tests (verbose)...$(NC)"
	@$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Generate test coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p coverage_output
	@$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_HTML)$(NC)"
	@echo "$(BLUE)Coverage summary:$(NC)"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total

.PHONY: test-race
test-race: ## Run tests with race condition detection
	@echo "$(GREEN)Running tests with race detection...$(NC)"
	@$(GOTEST) -race ./...

.PHONY: test-short
test-short: ## Run only short tests
	@echo "$(GREEN)Running short tests...$(NC)"
	@$(GOTEST) -short ./...

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

# =============================================================================
# Code Quality Targets
# =============================================================================

.PHONY: fmt
fmt: ## Format all Go code with gofmt
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) -w .
	@echo "$(GREEN)Code formatting complete!$(NC)"

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "$(RED)The following files need formatting:$(NC)"; \
		$(GOFMT) -l .; \
		exit 1; \
	else \
		echo "$(GREEN)All files are properly formatted!$(NC)"; \
	fi

.PHONY: vet
vet: ## Run go vet on all packages
	@echo "$(GREEN)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)Vet complete!$(NC)"

.PHONY: lint
lint: ## Run golangci-lint (must be installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(GREEN)Running golangci-lint...$(NC)"; \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Install with:$(NC)"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: tidy
tidy: ## Run go mod tidy to clean up dependencies
	@echo "$(GREEN)Tidying module dependencies...$(NC)"
	@$(GOMOD) tidy
	@echo "$(GREEN)Dependencies tidied!$(NC)"

.PHONY: check
check: fmt-check vet lint ## Run all code quality checks

# =============================================================================
# Git Hooks and CI Targets
# =============================================================================

.PHONY: install-hooks
install-hooks: ## Install git pre-commit and pre-push hooks
	@echo "$(GREEN)Installing git hooks...$(NC)"
	@./scripts/install-hooks.sh

.PHONY: pre-commit
pre-commit: fmt-check vet ## Run pre-commit checks (fast checks only)
	@echo "$(GREEN)Running pre-commit checks...$(NC)"
	@echo "$(GREEN)Checking test compilation...$(NC)"
	@$(GOTEST) -run=^$$ ./... > /dev/null 2>&1
	@echo "$(GREEN)All pre-commit checks passed!$(NC)"

.PHONY: pre-push
pre-push: fmt-check vet test test-race lint build ## Run pre-push checks (comprehensive)
	@echo "$(GREEN)All pre-push checks passed!$(NC)"

.PHONY: ci-local
ci-local: ## Run full CI suite locally (equivalent to GitHub Actions)
	@echo "$(BLUE)==============================================================$(NC)"
	@echo "$(BLUE)Running Full CI Suite Locally$(NC)"
	@echo "$(BLUE)==============================================================$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Format Check$(NC)"
	@$(MAKE) fmt-check
	@echo ""
	@echo "$(YELLOW)2. Vet Check$(NC)"
	@$(MAKE) vet
	@echo ""
	@echo "$(YELLOW)3. Lint Check$(NC)"
	@$(MAKE) lint
	@echo ""
	@echo "$(YELLOW)4. Unit Tests$(NC)"
	@$(MAKE) test
	@echo ""
	@echo "$(YELLOW)5. Race Detection$(NC)"
	@$(MAKE) test-race
	@echo ""
	@echo "$(YELLOW)6. Build$(NC)"
	@$(MAKE) build
	@echo ""
	@echo "$(GREEN)==============================================================$(NC)"
	@echo "$(GREEN)✅ All CI checks passed successfully!$(NC)"
	@echo "$(GREEN)==============================================================$(NC)"

# =============================================================================
# Development Targets
# =============================================================================

.PHONY: run
run: build ## Build and run the gor CLI (shows help)
	@$(BIN_DIR)/$(BINARY_NAME) --help

.PHONY: run-webapp
run-webapp: ## Run the webapp example
	@echo "$(GREEN)Starting webapp example...$(NC)"
	@$(GOCMD) run $(EXAMPLES_DIR)/webapp/main.go

.PHONY: run-auth
run-auth: ## Run the auth demo example
	@echo "$(GREEN)Starting auth demo...$(NC)"
	@$(GOCMD) run $(EXAMPLES_DIR)/auth_demo/main.go

.PHONY: run-blog
run-blog: ## Run the blog example
	@echo "$(GREEN)Starting blog example...$(NC)"
	@$(GOCMD) run $(EXAMPLES_DIR)/blog/main.go

.PHONY: run-realtime
run-realtime: ## Run the realtime demo
	@echo "$(GREEN)Starting realtime demo...$(NC)"
	@$(GOCMD) run $(EXAMPLES_DIR)/realtime_demo/main.go

.PHONY: run-solid
run-solid: ## Run the solid trifecta demo (Queue, Cache, Cable)
	@echo "$(GREEN)Starting solid trifecta demo...$(NC)"
	@$(GOCMD) run $(EXAMPLES_DIR)/solid_trifecta/main.go

.PHONY: dev
dev: build ## Start development server with the CLI
	@echo "$(GREEN)Starting development server...$(NC)"
	@$(BIN_DIR)/$(BINARY_NAME) server

.PHONY: console
console: build ## Start interactive console
	@echo "$(GREEN)Starting interactive console...$(NC)"
	@$(BIN_DIR)/$(BINARY_NAME) console

# =============================================================================
# Database Targets
# =============================================================================

.PHONY: db-clean
db-clean: ## Clean all database files
	@echo "$(YELLOW)Cleaning database files...$(NC)"
	@find . -name "*.db" -type f -delete 2>/dev/null || true
	@find . -name "*.db-journal" -type f -delete 2>/dev/null || true
	@find . -name "*.sqlite" -type f -delete 2>/dev/null || true
	@find . -name "*.sqlite3" -type f -delete 2>/dev/null || true
	@echo "$(GREEN)Database files cleaned!$(NC)"

.PHONY: migrate
migrate: build ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@$(BIN_DIR)/$(BINARY_NAME) db migrate

# =============================================================================
# Documentation Targets
# =============================================================================

.PHONY: docs
docs: ## Generate documentation
	@echo "$(GREEN)Generating documentation...$(NC)"
	@$(GOCMD) doc -all ./pkg/gor > docs/api-reference.txt
	@echo "$(GREEN)Documentation generated!$(NC)"

# =============================================================================
# CI/CD Targets
# =============================================================================

.PHONY: ci
ci: tidy fmt-check vet test build ## Run full CI pipeline
	@echo "$(GREEN)CI pipeline complete!$(NC)"

.PHONY: release
release: clean ## Build release binaries for multiple platforms
	@echo "$(GREEN)Building release binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)

	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

	@echo "Building for Darwin AMD64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

	@echo "Building for Darwin ARM64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

	@echo "$(GREEN)Release binaries built in $(BUILD_DIR)$(NC)"

# =============================================================================
# Utility Targets
# =============================================================================

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) verify
	@echo "$(GREEN)Dependencies downloaded and verified!$(NC)"

.PHONY: update-deps
update-deps: ## Update all dependencies to latest versions
	@echo "$(GREEN)Updating dependencies...$(NC)"
	@$(GOCMD) get -u ./...
	@$(GOMOD) tidy
	@echo "$(GREEN)Dependencies updated!$(NC)"

.PHONY: tools
tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOCMD) install golang.org/x/tools/cmd/goimports@latest
	@$(GOCMD) install github.com/cosmtrek/air@latest
	@echo "$(GREEN)Development tools installed!$(NC)"

# =============================================================================
# Help Target
# =============================================================================

.PHONY: help
help: ## Display this help message
	@echo "$(BLUE)Gor Framework - Make Targets$(NC)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "$(YELLOW)Available targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)Examples:$(NC)"
	@echo "  make build          # Build the gor CLI"
	@echo "  make test           # Run all tests"
	@echo "  make run-webapp     # Run the webapp example"
	@echo "  make ci             # Run full CI pipeline"
	@echo ""
	@echo "$(BLUE)Quick Start:$(NC)"
	@echo "  1. make deps        # Download dependencies"
	@echo "  2. make build       # Build the CLI"
	@echo "  3. make test        # Run tests"
	@echo "  4. make run-webapp  # Try an example"