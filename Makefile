.PHONY: help build test test-coverage lint fmt vet clean install-tools check publish-check tag-release

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Packages
PACKAGES := . ./thru/dynamodb ./thru/sql ./thru/exprlang ./thru/jsonapi
ALL_PACKAGES := $(shell go list ./...)

# Version (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

help: ## Display this help message
	@echo "Sift - Universal Query Library for Go"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build all packages
	@echo "Building packages..."
	@for pkg in $(PACKAGES); do \
		echo "  Building $$pkg..."; \
		cd $$pkg && $(GOBUILD) -v ./... && cd - > /dev/null || exit 1; \
	done
	@echo "✓ Build complete"

test: ## Run tests for all packages
	@echo "Running tests..."
	$(GOTEST) -v -race -timeout 30s ./...
	@echo "✓ Tests passed"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	$(GOTEST) -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "✓ Coverage report generated: coverage/coverage.html"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep total | awk '{print "  Total coverage: " $$3}'

test-short: ## Run tests without race detector (faster)
	@echo "Running tests (short mode)..."
	$(GOTEST) -v -short ./...
	@echo "✓ Tests passed"

lint: install-tools ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
		echo "✓ Linting complete"; \
	else \
		echo "⚠ golangci-lint not found. Install with: make install-tools"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "✓ Vet complete"

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "✓ All checks passed"

clean: ## Clean build artifacts and caches
	@echo "Cleaning..."
	@rm -rf coverage/
	@$(GOCMD) clean -cache -testcache -modcache
	@echo "✓ Clean complete"

tidy: ## Tidy and verify go.mod files
	@echo "Tidying go.mod files..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Tidying $$pkg/go.mod..."; \
			cd $$pkg && $(GOMOD) tidy && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo "✓ Tidy complete"

verify: tidy ## Verify dependencies
	@echo "Verifying dependencies..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Verifying $$pkg..."; \
			cd $$pkg && $(GOMOD) verify && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo "✓ Verification complete"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "  Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	else \
		echo "  golangci-lint already installed"; \
	fi
	@echo "✓ Tools installed"

publish-check: ## Check if packages are ready for publishing
	@echo "Checking packages for publishing..."
	@echo ""
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "Checking git status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "✗ Working directory is not clean. Commit or stash changes."; \
		exit 1; \
	else \
		echo "✓ Working directory is clean"; \
	fi
	@echo ""
	@echo "Running tests..."
	@$(MAKE) test-short
	@echo ""
	@echo "Checking go.mod files..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Checking $$pkg/go.mod..."; \
			cd $$pkg && $(GOMOD) tidy && cd - > /dev/null || exit 1; \
			if [ -n "$$(git diff $$pkg/go.mod)" ]; then \
				echo "✗ $$pkg/go.mod needs tidying"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "✓ All go.mod files are tidy"
	@echo ""
	@echo "✓ Packages are ready for publishing"
	@echo ""
	@echo "To publish, create and push a git tag:"
	@echo "  git tag v$(VERSION)"
	@echo "  git push origin v$(VERSION)"
	@echo ""
	@echo "For submodules, use module-specific tags:"
	@echo "  git tag thru/dynamodb/v$(VERSION)"
	@echo "  git tag thru/sql/v$(VERSION)"
	@echo "  git tag thru/exprlang/v$(VERSION)"
	@echo "  git tag thru/jsonapi/v$(VERSION)"

tag-release: ## Create git tags for release (VERSION=x.y.z required)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "✗ VERSION is required. Usage: make tag-release VERSION=1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release tags for version $(VERSION)..."
	@echo ""
	@echo "Main module:"
	@git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	@echo "  Created tag: v$(VERSION)"
	@echo ""
	@echo "Submodules:"
	@git tag -a "thru/dynamodb/v$(VERSION)" -m "Release thru/dynamodb v$(VERSION)"
	@echo "  Created tag: thru/dynamodb/v$(VERSION)"
	@git tag -a "thru/sql/v$(VERSION)" -m "Release thru/sql v$(VERSION)"
	@echo "  Created tag: thru/sql/v$(VERSION)"
	@git tag -a "thru/exprlang/v$(VERSION)" -m "Release thru/exprlang v$(VERSION)"
	@echo "  Created tag: thru/exprlang/v$(VERSION)"
	@git tag -a "thru/jsonapi/v$(VERSION)" -m "Release thru/jsonapi v$(VERSION)"
	@echo "  Created tag: thru/jsonapi/v$(VERSION)"
	@echo ""
	@echo "✓ Tags created successfully"
	@echo ""
	@echo "To push tags to remote:"
	@echo "  git push origin --tags"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Downloading for $$pkg..."; \
			cd $$pkg && $(GOMOD) download && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo "✓ Dependencies downloaded"

list-packages: ## List all packages
	@echo "Packages:"
	@for pkg in $(PACKAGES); do \
		echo "  $$pkg"; \
	done

version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Git branch: $$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"
	@echo "Go version: $$(go version)"
