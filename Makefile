.PHONY: help build test test-coverage lint fmt vet clean install-tools check publish-check publish-prepare publish publish-restore publish-proxy tag-release list-tags delete-tag release-info changelog

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
	@echo "Build complete"

test: ## Run tests for all packages
	@echo "Running tests..."
	$(GOTEST) -v -race -timeout 30s ./...
	@echo "Tests passed"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	$(GOTEST) -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep total | awk '{print "  Total coverage: " $$3}'

test-short: ## Run tests without race detector (faster)
	@echo "Running tests (short mode)..."
	$(GOTEST) -v -short ./...
	@echo "Tests passed"

lint: install-tools ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
		echo "Linting complete"; \
	else \
		echo "Warning: golangci-lint not found. Install with: make install-tools"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "All checks passed"

clean: ## Clean build artifacts and caches
	@echo "Cleaning..."
	@rm -rf coverage/
	@$(GOCMD) clean -cache -testcache -modcache
	@echo "Clean complete"

tidy: ## Tidy and verify go.mod files
	@echo "Tidying go.mod files..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Tidying $$pkg/go.mod..."; \
			cd $$pkg && $(GOMOD) tidy && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo "Tidy complete"

verify: tidy ## Verify dependencies
	@echo "Verifying dependencies..."
	@for pkg in $(PACKAGES); do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Verifying $$pkg..."; \
			cd $$pkg && $(GOMOD) verify && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo "Verification complete"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "  Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	else \
		echo "  golangci-lint already installed"; \
	fi
	@echo "Tools installed"

publish-check: ## Check if packages are ready for publishing
	@echo "Checking packages for publishing..."
	@echo ""
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "Checking git status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory is not clean. Commit or stash changes."; \
		exit 1; \
	else \
		echo "Working directory is clean"; \
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
				echo "Error: $$pkg/go.mod needs tidying"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "All go.mod files are tidy"
	@echo ""
	@echo "Packages are ready for publishing"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run: make publish-prepare VERSION=x.y.z"
	@echo "  2. Review and commit the go.mod changes"
	@echo "  3. Create and push tags: make tag-release VERSION=x.y.z"

publish-prepare: ## Prepare submodules for publishing (VERSION=x.y.z required)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION is required. Usage: make publish-prepare VERSION=1.0.0"; \
		exit 1; \
	fi
	@echo "Preparing submodules for release v$(VERSION)..."
	@echo ""
	@echo "Updating submodule go.mod files..."
	@for pkg in ./thru/dynamodb ./thru/sql ./thru/exprlang ./thru/jsonapi; do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Updating $$pkg/go.mod..."; \
			sed -i.bak 's|require github.com/nisimpson/sift v.*|require github.com/nisimpson/sift v$(VERSION)|' $$pkg/go.mod; \
			rm -f $$pkg/go.mod.bak; \
			cd $$pkg && $(GOMOD) tidy && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo ""
	@echo "✅ Submodules prepared for release v$(VERSION)"
	@echo ""
	@echo "Changes made:"
	@git diff --stat
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review changes: git diff"
	@echo "  2. Commit changes: git add . && git commit -m 'chore: prepare for v$(VERSION) release'"
	@echo "  3. Create tags: make tag-release VERSION=$(VERSION)"
	@echo "  4. Push: git push origin main --tags"

tag-release: ## Create git tags for release (VERSION=x.y.z required, MODULE=path optional)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION is required. Usage: make tag-release VERSION=1.0.0"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make tag-release VERSION=1.0.0                    # Tag all modules"; \
		echo "  make tag-release VERSION=1.0.0 MODULE=.           # Tag main module only"; \
		echo "  make tag-release VERSION=1.0.1 MODULE=thru/sql    # Tag SQL adapter only"; \
		exit 1; \
	fi
	@if [ -n "$(MODULE)" ]; then \
		if [ "$(MODULE)" = "." ]; then \
			echo "Creating tag for main module..."; \
			git tag -a "v$(VERSION)" -m "Release v$(VERSION)"; \
			echo "Created tag: v$(VERSION)"; \
		else \
			echo "Creating tag for $(MODULE)..."; \
			git tag -a "$(MODULE)/v$(VERSION)" -m "Release $(MODULE) v$(VERSION)"; \
			echo "Created tag: $(MODULE)/v$(VERSION)"; \
		fi; \
	else \
		echo "Creating release tags for ALL modules (version $(VERSION))..."; \
		echo ""; \
		echo "Main module:"; \
		git tag -a "v$(VERSION)" -m "Release v$(VERSION)"; \
		echo "  Created tag: v$(VERSION)"; \
		echo ""; \
		echo "Submodules:"; \
		git tag -a "thru/dynamodb/v$(VERSION)" -m "Release thru/dynamodb v$(VERSION)"; \
		echo "  Created tag: thru/dynamodb/v$(VERSION)"; \
		git tag -a "thru/sql/v$(VERSION)" -m "Release thru/sql v$(VERSION)"; \
		echo "  Created tag: thru/sql/v$(VERSION)"; \
		git tag -a "thru/exprlang/v$(VERSION)" -m "Release thru/exprlang v$(VERSION)"; \
		echo "  Created tag: thru/exprlang/v$(VERSION)"; \
		git tag -a "thru/jsonapi/v$(VERSION)" -m "Release thru/jsonapi v$(VERSION)"; \
		echo "  Created tag: thru/jsonapi/v$(VERSION)"; \
		echo ""; \
		echo "All tags created successfully"; \
	fi
	@echo ""
	@echo "To push tags to remote:"
	@echo "  git push origin --tags"
	@echo ""
	@echo "Or push specific tag:"
	@if [ -n "$(MODULE)" ]; then \
		if [ "$(MODULE)" = "." ]; then \
			echo "  git push origin v$(VERSION)"; \
		else \
			echo "  git push origin $(MODULE)/v$(VERSION)"; \
		fi; \
	fi

list-tags: ## List all version tags
	@echo "All version tags:"
	@git tag -l | grep -E '^v[0-9]|^thru/' | sort -V || echo "  No tags found"

delete-tag: ## Delete a tag (TAG=v1.0.0 or TAG=thru/sql/v1.0.0 required)
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required. Usage: make delete-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "Deleting tag: $(TAG)"
	@git tag -d "$(TAG)"
	@echo "Local tag deleted"
	@echo ""
	@echo "To delete from remote:"
	@echo "  git push origin :refs/tags/$(TAG)"

release-info: ## Show release info for a tag (TAG=v1.0.0 required)
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required. Usage: make release-info TAG=v1.0.0"; \
		exit 1; \
	fi
	@./scripts/release-info.sh "$(TAG)"

changelog: ## Generate changelog for a tag (TAG=v1.0.0 required)
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required. Usage: make changelog TAG=v1.0.0"; \
		exit 1; \
	fi
	@MODULE_PATH=$$(./scripts/release-info.sh "$(TAG)" | grep MODULE_PATH | cut -d= -f2); \
	./scripts/generate-changelog.sh "$(TAG)" "$$MODULE_PATH"

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
	@echo "Dependencies downloaded"

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

publish: ## Publish a new release (VERSION=x.y.z required, MODULE=path optional)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION is required. Usage: make publish VERSION=1.0.0"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make publish VERSION=1.0.0                    # Publish all modules"; \
		echo "  make publish VERSION=1.0.0 MODULE=.           # Publish main module only"; \
		echo "  make publish VERSION=1.0.1 MODULE=thru/sql    # Publish SQL adapter only"; \
		exit 1; \
	fi
	@echo "Publishing release v$(VERSION)..."
	@echo ""
	@echo "Step 1: Running pre-publish checks..."
	@$(MAKE) publish-check
	@echo ""
	@echo "Step 2: Preparing submodules..."
	@$(MAKE) publish-prepare VERSION=$(VERSION)
	@echo ""
	@echo "Step 3: Committing version changes..."
	@git add .
	@git commit -m "chore: prepare for v$(VERSION) release"
	@echo ""
	@echo "Step 4: Creating git tags..."
	@if [ -n "$(MODULE)" ]; then \
		$(MAKE) tag-release VERSION=$(VERSION) MODULE=$(MODULE); \
	else \
		$(MAKE) tag-release VERSION=$(VERSION); \
	fi
	@echo ""
	@echo "Step 5: Pushing to remote..."
	@git push origin main
	@git push origin --tags
	@echo ""
	@echo "✅ Release v$(VERSION) published successfully!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Trigger Go proxy indexing: make publish-proxy VERSION=$(VERSION)"
	@echo "  2. Verify on GitHub: https://github.com/nisimpson/sift/releases"
	@echo "  3. Check pkg.go.dev (may take a few minutes to index)"
	@echo "  4. Restore development versions: make publish-restore"

publish-restore: ## Restore v0.0.0 versions for development
	@echo "Restoring development versions..."
	@echo ""
	@for pkg in ./thru/dynamodb ./thru/sql ./thru/exprlang ./thru/jsonapi; do \
		if [ -f "$$pkg/go.mod" ]; then \
			echo "  Updating $$pkg/go.mod..."; \
			sed -i.bak 's|require github.com/nisimpson/sift v.*|require github.com/nisimpson/sift v0.0.0|' $$pkg/go.mod; \
			rm -f $$pkg/go.mod.bak; \
			cd $$pkg && $(GOMOD) tidy && cd - > /dev/null || exit 1; \
		fi; \
	done
	@echo ""
	@echo "✅ Development versions restored"
	@echo ""
	@git add .
	@git commit -m "chore: restore v0.0.0 for development"
	@git push origin main
	@echo ""
	@echo "Ready for development!"

publish-proxy: ## Trigger Go module proxy to index published modules (VERSION=x.y.z required, MODULE=path optional)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION is required. Usage: make publish-proxy VERSION=1.0.0"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make publish-proxy VERSION=1.0.0                    # Index all modules"; \
		echo "  make publish-proxy VERSION=1.0.0 MODULE=.           # Index main module only"; \
		echo "  make publish-proxy VERSION=1.0.1 MODULE=thru/sql    # Index SQL adapter only"; \
		exit 1; \
	fi
	@echo "Triggering Go module proxy to index modules..."
	@echo ""
	@if [ -n "$(MODULE)" ]; then \
		if [ "$(MODULE)" = "." ]; then \
			echo "Indexing main module v$(VERSION)..."; \
			GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift@v$(VERSION) || true; \
		else \
			echo "Indexing $(MODULE) v$(VERSION)..."; \
			GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift/$(MODULE)@v$(VERSION) || true; \
		fi; \
	else \
		echo "Indexing main module v$(VERSION)..."; \
		GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift@v$(VERSION) || true; \
		echo ""; \
		echo "Indexing thru/dynamodb v$(VERSION)..."; \
		GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift/thru/dynamodb@v$(VERSION) || true; \
		echo ""; \
		echo "Indexing thru/sql v$(VERSION)..."; \
		GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift/thru/sql@v$(VERSION) || true; \
		echo ""; \
		echo "Indexing thru/exprlang v$(VERSION)..."; \
		GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift/thru/exprlang@v$(VERSION) || true; \
		echo ""; \
		echo "Indexing thru/jsonapi v$(VERSION)..."; \
		GOPROXY=proxy.golang.org go list -m github.com/nisimpson/sift/thru/jsonapi@v$(VERSION) || true; \
	fi
	@echo ""
	@echo "✅ Proxy indexing triggered"
	@echo ""
	@echo "Verify on pkg.go.dev (may take a few minutes):"
	@if [ -n "$(MODULE)" ]; then \
		if [ "$(MODULE)" = "." ]; then \
			echo "  https://pkg.go.dev/github.com/nisimpson/sift@v$(VERSION)"; \
		else \
			echo "  https://pkg.go.dev/github.com/nisimpson/sift/$(MODULE)@v$(VERSION)"; \
		fi; \
	else \
		echo "  https://pkg.go.dev/github.com/nisimpson/sift@v$(VERSION)"; \
		echo "  https://pkg.go.dev/github.com/nisimpson/sift/thru/dynamodb@v$(VERSION)"; \
		echo "  https://pkg.go.dev/github.com/nisimpson/sift/thru/sql@v$(VERSION)"; \
		echo "  https://pkg.go.dev/github.com/nisimpson/sift/thru/exprlang@v$(VERSION)"; \
		echo "  https://pkg.go.dev/github.com/nisimpson/sift/thru/jsonapi@v$(VERSION)"; \
	fi
