# Dolly - Advanced Tmux Session Manager
# A powerful YAML-based tmux session manager with terminal shell support and pre-hooks

.PHONY: all build install clean test help deps lint vet fmt check run-sample run-distill dev-setup

# Default target
all: clean deps build test

# Variables
BINARY_NAME=dolly
TEST_BINARY=test_runner
INSTALL_PATH=/usr/local/bin
VERSION=v1.0.0

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)Dolly - Advanced Tmux Session Manager$(NC)"
	@echo "$(YELLOW)Available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

deps: ## Install dependencies
	@echo "$(YELLOW)Installing dependencies...$(NC)"
	go mod tidy
	go mod download

build: ## Build the dolly binary
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) main.go
	@echo "$(GREEN)✅ Built $(BINARY_NAME)$(NC)"

build-test: ## Build the test runner
	@echo "$(YELLOW)Building test runner...$(NC)"
	go build -o $(TEST_BINARY) ./cmd/test
	@echo "$(GREEN)✅ Built $(TEST_BINARY)$(NC)"

install: build ## Install dolly to system PATH
	@echo "$(YELLOW)Installing $(BINARY_NAME) to $(INSTALL_PATH)...$(NC)"
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(GREEN)✅ Installed $(BINARY_NAME) to $(INSTALL_PATH)$(NC)"
	@echo "$(BLUE)You can now run 'dolly' from anywhere!$(NC)"

uninstall: ## Uninstall dolly from system PATH
	@echo "$(YELLOW)Uninstalling $(BINARY_NAME) from $(INSTALL_PATH)...$(NC)"
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "$(GREEN)✅ Uninstalled $(BINARY_NAME)$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -f $(BINARY_NAME) $(TEST_BINARY)
	go clean
	@echo "$(GREEN)✅ Cleaned$(NC)"

test: build-test ## Run the comprehensive test suite
	@echo "$(YELLOW)Running test suite...$(NC)"
	./$(TEST_BINARY)
	@echo "$(GREEN)✅ All tests passed$(NC)"

lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)✅ go vet passed$(NC)"

fmt: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

check: fmt vet lint ## Run all code quality checks
	@echo "$(GREEN)✅ All checks passed$(NC)"

run-sample: build ## Run with sample configuration
	@echo "$(YELLOW)Running dolly with sample configuration...$(NC)"
	./$(BINARY_NAME) sample-config.yml

run-distill: build ## Run with distill configuration
	@echo "$(YELLOW)Running dolly with distill configuration...$(NC)"
	./$(BINARY_NAME) distill-config.yml

dev-setup: deps ## Set up development environment
	@echo "$(YELLOW)Setting up development environment...$(NC)"
	@if ! command -v golangci-lint > /dev/null; then \
		echo "$(BLUE)Installing golangci-lint...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "$(GREEN)✅ Development environment ready$(NC)"

release: clean check build test ## Prepare release build
	@echo "$(YELLOW)Preparing release build...$(NC)"
	@echo "$(GREEN)✅ Release ready: $(BINARY_NAME) $(VERSION)$(NC)"

version: ## Show version information
	@echo "$(BLUE)Dolly $(VERSION)$(NC)"
	@echo "Go version: $(shell go version)"
	@echo "Build date: $(shell date)"

examples: ## Show usage examples
	@echo "$(BLUE)Dolly Usage Examples:$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Basic usage:$(NC)"
	@echo "   ./dolly my-config.yml"
	@echo ""
	@echo "$(YELLOW)2. Run with sample configuration:$(NC)"
	@echo "   make run-sample"
	@echo ""
	@echo "$(YELLOW)3. Create your own config:$(NC)"
	@echo "   cp sample-config.yml my-project.yml"
	@echo "   # Edit my-project.yml with your settings"
	@echo "   ./dolly my-project.yml"
	@echo ""
	@echo "$(YELLOW)4. Test the setup:$(NC)"
	@echo "   make test"

quick-start: build ## Quick start guide
	@echo "$(BLUE)🚀 Dolly Quick Start:$(NC)"
	@echo ""
	@echo "$(GREEN)1.$(NC) Copy and customize the sample config:"
	@echo "   cp sample-config.yml my-project.yml"
	@echo ""
	@echo "$(GREEN)2.$(NC) Edit my-project.yml with your project settings"
	@echo ""
	@echo "$(GREEN)3.$(NC) Run dolly:"
	@echo "   ./dolly my-project.yml"
	@echo ""
	@echo "$(GREEN)4.$(NC) Your tmux session is ready! 🎉"
	@echo ""
	@echo "$(YELLOW)For more examples: make examples$(NC)"

# Development targets
.PHONY: watch
watch: ## Watch for changes and rebuild (requires entr)
	@if command -v entr > /dev/null; then \
		find . -name '*.go' | entr -r make build; \
	else \
		echo "$(RED)entr not found. Install with: brew install entr (macOS) or apt-get install entr (Linux)$(NC)"; \
	fi