# picosend Makefile

# Variables
BINARY_NAME=picosend
PORT?=8080
GO=go
DOCKER=docker

# Go build variables
LDFLAGS=-ldflags "-s -w -X main.Version=$(shell git describe --tags --always 2>/dev/null || echo 'dev')"

.PHONY: help
help: ## Show this help message
	@echo 'picosend - One-time secret sharing'
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the application (default port 8080)
	$(GO) run .

.PHONY: run-dev
run-dev: PORT?=8081
run-dev: ## Run on development port 8081
	PORT=$(PORT) $(GO) run .

.PHONY: build
build: ## Build the binary
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) .

.PHONY: build-linux
build-linux: ## Build for Linux (amd64)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .

.PHONY: build-darwin
build-darwin: ## Build for macOS (amd64)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .

.PHONY: build-all
build-all: build-linux build-darwin ## Build binaries for all platforms

.PHONY: test
test: ## Run all tests
	$(GO) test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	$(GO) test -cover ./...

.PHONY: test-coverage-html
test-coverage-html: ## Run tests with coverage report (HTML)
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: test-unit
test-unit: ## Run unit tests only
	$(GO) test -v ./... -run "Test.*Store|TestGenerateID"

.PHONY: test-handler
test-handler: ## Run handler tests only
	$(GO) test -v ./... -run "Test.*Handler"

.PHONY: test-integration
test-integration: ## Run integration tests only
	$(GO) test -v ./... -run "TestFull|TestDirect|TestHome|TestView|TestConcurrent"

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GO) test -race -v ./...

.PHONY: test-bench
test-bench: ## Run benchmarks
	$(GO) test -bench=. ./...

.PHONY: test-full
test-full: ## Run comprehensive test suite (same as ./test.sh)
	./test.sh

.PHONY: fmt
fmt: ## Format Go code
	$(GO) fmt ./...

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@! $(GO) fmt ./... | grep -q '.*\.go'

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run all static checks (fmt, vet)
	@echo "Running go fmt..."
	$(GO) fmt ./...
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "All checks passed!"

.PHONY: tidy
tidy: ## Clean up dependencies
	$(GO) mod tidy

.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download

.PHONY: clean
clean: ## Clean build artifacts
	$(GO) clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

.PHONY: docker-build
docker-build: ## Build Docker image
	$(DOCKER) build -t picosend:latest .

.PHONY: docker-run
docker-run: ## Run Docker container (port 8080)
	$(DOCKER) run -p 8080:8080 picosend:latest

.PHONY: docker-run-dev
docker-run-dev: ## Run Docker container (port 8081)
	$(DOCKER) run -p 8081:8081 -e PORT=8081 picosend:latest

.PHONY: docker-clean
docker-clean: ## Remove Docker image
	$(DOCKER) rmi picosend:latest

.PHONY: install
install: ## Install the binary to GOPATH/bin
	$(GO) install $(LDFLAGS) .

.PHONY: ci
ci: fmt-check vet test ## Run CI checks (format, vet, test)

.DEFAULT_GOAL := help
