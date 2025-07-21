VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%d-%H:%M:%S")

# Build settings
BUILD_DIR := bin
BINARY_NAME := testrigor-ci-tool
COVERAGE_DIR := $(BUILD_DIR)/coverage

# Go settings
GO := go
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: install
install: build ## Install the application to GOBIN
	@echo "Installing $(BINARY_NAME)..."
	install_dir=$${GOBIN:-$$(go env GOPATH)/bin}; \
	mkdir -p $$install_dir; \
	cp $(BUILD_DIR)/$(BINARY_NAME) $$install_dir/

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	goimports -w .

.PHONY: lint
lint: ## Run linting checks
	@echo "Running linting checks..."
	$(GO) vet ./...
	golangci-lint run

.PHONY: test
test: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

.PHONY: test-short
test-short: ## Run tests without coverage (faster)
	@echo "Running tests (short mode)..."
	$(GO) test -v -short ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	$(GO) test -v -race ./...

.PHONY: test-coverage
test-coverage: test ## Generate and display test coverage report
	@echo "Test coverage report:"
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo ""
	@echo "HTML coverage report: $(COVERAGE_DIR)/coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify

.PHONY: tidy
tidy: ## Tidy up go.mod and go.sum
	@echo "Tidying up dependencies..."
	$(GO) mod tidy

.PHONY: vendor
vendor: ## Create vendor directory
	@echo "Creating vendor directory..."
	$(GO) mod vendor

.PHONY: check
check: fmt lint test ## Run all quality checks
	@echo "All quality checks passed!"

.PHONY: ci
ci: deps check build ## Run CI pipeline locally
	@echo "CI pipeline completed successfully!"

.PHONY: security
security: ## Run security checks
	@echo "Running security checks..."
	govulncheck ./...

.PHONY: doc
doc: ## Generate and serve documentation
	@echo "Starting documentation server..."
	@echo "Open http://localhost:6060/pkg/github.com/benvon/testrigor-ci-tool/ in your browser"
	godoc -http=:6060

.PHONY: generate
generate: ## Run go generate
	@echo "Running go generate..."
	$(GO) generate ./...

.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: release-dry-run
release-dry-run: ## Test release process without publishing
	@echo "Running release dry-run..."
	goreleaser release --snapshot --clean

.PHONY: all
all: check build ## Build everything and run all checks

# Development targets
.PHONY: dev-setup
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Development environment setup complete!"

.PHONY: watch
watch: ## Watch for changes and run tests
	@echo "Watching for changes..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	fswatch -o . | xargs -n1 -I{} make test-short 