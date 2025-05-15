VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%d-%H:%M:%S")

.PHONY: build
build:
	mkdir -p bin
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/testrigor-ci-tool

.PHONY: install
install: build
	install_dir=$${GOBIN:-$$(go env GOPATH)/bin}; \
	mkdir -p $$install_dir; \
	cp bin/testrigor-ci-tool $$install_dir/

.PHONY: test
test:
	mkdir -p bin
	go test -v -race -coverprofile=bin/coverage.out ./... || exit 1
	[ -f bin/coverage.out ] && go tool cover -html=bin/coverage.out -o bin/coverage.html || true

.PHONY: test-short
test-short:
	go test -v -short ./...

.PHONY: test-race
test-race:
	go test -v -race ./...

.PHONY: test-coverage
test-coverage:
	mkdir -p bin
	go test -v -coverprofile=bin/coverage.out ./... || exit 1
	[ -f bin/coverage.out ] && go tool cover -html=bin/coverage.out -o bin/coverage.html || true

.PHONY: clean
clean:
	rm -rf bin/ 