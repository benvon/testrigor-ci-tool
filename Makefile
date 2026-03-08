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
fmt: ## Format Go code (fix formatting)
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .

.PHONY: fmt-check
fmt-check: ## Verify Go formatting (CI check - no modifications)
	@echo "Checking Go formatting..."
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
	  echo "The following files are not properly formatted (run 'make fmt' to fix):"; \
	  echo "$$unformatted"; \
	  exit 1; \
	fi
	@unformatted=$$($(GO) run golang.org/x/tools/cmd/goimports@latest -l .); \
	if [ -n "$$unformatted" ]; then \
	  echo "The following files have incorrect imports (run 'make fmt' to fix):"; \
	  echo "$$unformatted"; \
	  exit 1; \
	fi

.PHONY: lint
lint: ## Run linting checks (aligned with CI)
	@echo "Running linting checks..."
	$(GO) vet ./...
	@echo "Checking go mod tidy..."
	@$(GO) mod tidy; \
	tidy_changes=$$(git diff --name-only go.mod go.sum 2>/dev/null); \
	if [ -n "$$tidy_changes" ]; then \
	  echo "go mod tidy resulted in changes. Please commit the changes to go.mod and go.sum:"; \
	  git diff go.mod go.sum; \
	  exit 1; \
	fi
	@echo "Running golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run --timeout=5m --verbose
	@echo "Running security check (gosec)..."
	@command -v gosec >/dev/null 2>&1 || $(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@PATH="$$(go env GOPATH)/bin:$$PATH" gosec ./...

.PHONY: test
test: ## Run tests with coverage (CI: requires >= 70% coverage)
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Checking test coverage (minimum 70%)..."
	@coverage=$$($(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Test coverage: $${coverage}%"; \
	if [ -n "$$coverage" ] && [ "$$(echo "$$coverage < 70" | bc 2>/dev/null)" = "1" ]; then \
	  echo "Test coverage is below 70% (minimum required)"; \
	  exit 1; \
	fi

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
check: fmt-check lint test ## Run all quality checks (aligned with CI - verify only)
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
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Development environment setup complete!"

.PHONY: watch
watch: ## Watch for changes and run tests
	@echo "Watching for changes..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	fswatch -o . | xargs -n1 -I{} make test-short 

.PHONY: validate
validate: ## Run lint, format, security checks, and ensure test coverage > 70% for PR validation
	@echo "Validating code for PR workflow..."
	golangci-lint run
	gofmt -l -s . | tee /dev/stderr | (! grep .)
	@command -v gosec >/dev/null 2>&1 || $(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...
	@echo "Running tests and checking coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@totalcov=$$(go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total: | sed 's/.* \\([0-9.]*\\)%/\\1/'); \
	if [ -z "$$totalcov" ]; then \
	  echo "Could not determine total coverage!"; \
	  exit 1; \
	fi; \
	lowcov=$$(echo "$$totalcov < 70.0" | bc); \
	if [ "$$lowcov" -eq 1 ]; then \
	  echo "Test coverage is too low: $$totalcov% (minimum is 70%)"; \
	  exit 1; \
	else \
	  echo "Test coverage is sufficient: $$totalcov%"; \
	fi
	@echo "Validation complete!" 