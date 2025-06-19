# Scanfrog Makefile
# Run 'make help' to see all available targets

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: fmt vet tidy lint test build ## Run all checks (CI runs this)

# Code Quality
.PHONY: fmt
fmt: ## Format code with go fmt
	@echo "Running go fmt..."
	@go fmt ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy and verify
	@echo "Running go mod tidy..."
	@go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod or go.sum is dirty after tidy" && exit 1)

# Building
.PHONY: build
build: ## Build binary for current platform
	@echo "Building scanfrog..."
	@go build -o scanfrog ./cmd/scanfrog

.PHONY: build-all
build-all: ## Build for all supported platforms
	@echo "Building for all platforms..."
	@GOOS=linux GOARCH=amd64 go build -o dist/scanfrog-linux-amd64 ./cmd/scanfrog
	@GOOS=linux GOARCH=arm64 go build -o dist/scanfrog-linux-arm64 ./cmd/scanfrog
	@GOOS=darwin GOARCH=amd64 go build -o dist/scanfrog-darwin-amd64 ./cmd/scanfrog
	@GOOS=darwin GOARCH=arm64 go build -o dist/scanfrog-darwin-arm64 ./cmd/scanfrog
	@GOOS=windows GOARCH=amd64 go build -o dist/scanfrog-windows-amd64.exe ./cmd/scanfrog

.PHONY: install
install: ## Install binary to GOPATH/bin
	@echo "Installing scanfrog..."
	@go install ./cmd/scanfrog

# Testing
.PHONY: test
test: ## Run all unit tests
	@echo "Running tests..."
	@go test ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-coverage-report
test-coverage-report: test-coverage ## Show coverage in terminal
	@go tool cover -func=coverage.out

.PHONY: test-integration
test-integration: build ## Run integration tests with sample data
	@echo "Running integration tests..."
	@./scanfrog --json testdata/sample-vulns.json --test-mode || true

.PHONY: test-interactive
test-interactive: ## Run terminal interaction tests
	@echo "Running interactive tests..."
	@go test -tags=interactive ./internal/game/...

# Security & Compliance
.PHONY: security
security: ## Run gosec security scanner
	@echo "Running security scan..."
	@gosec -quiet ./...

.PHONY: licenses
licenses: ## Check dependency licenses
	@echo "Checking licenses..."
	@go-licenses check ./cmd/scanfrog

.PHONY: vulncheck
vulncheck: ## Check for vulnerable dependencies
	@echo "Checking for vulnerabilities..."
	@govulncheck ./...

# Game-Specific
.PHONY: test-render
test-render: build ## Test terminal rendering
	@echo "Testing terminal rendering..."
	@./scanfrog --json testdata/sample-vulns.json --test-render || true

.PHONY: smoke-test
smoke-test: build ## Run binary with test data as smoke test
	@echo "Running smoke test..."
	@timeout 5s ./scanfrog --json testdata/sample-vulns.json || true

# Release
.PHONY: release-dry-run
release-dry-run: ## Test goreleaser without publishing
	@echo "Running release dry run..."
	@goreleaser release --snapshot --clean

.PHONY: release-snapshot
release-snapshot: ## Create release artifacts locally
	@echo "Creating release snapshot..."
	@goreleaser release --snapshot --clean

# Development Helpers
.PHONY: clean
clean: ## Remove built binaries and test artifacts
	@echo "Cleaning..."
	@rm -f scanfrog
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@go clean -testcache

.PHONY: setup
setup: ## Install required development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/google/go-licenses@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/goreleaser/goreleaser@latest
	@echo "All tools installed!"

.PHONY: pre-commit
pre-commit: fmt vet tidy test ## Run quick checks before committing
	@echo "Pre-commit checks passed!"

# Default target
.DEFAULT_GOAL := help