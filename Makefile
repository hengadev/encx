# Makefile for encx project with code generation integration

.PHONY: help generate build test lint clean install-tools validate

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Code generation targets
generate: ## Generate encx code for all packages
	@echo "Generating encx code..."
	@go generate ./...
	@echo "Code generation complete!"

generate-verbose: ## Generate encx code with verbose output
	@echo "Generating encx code (verbose)..."
	@ENCX_VERBOSE=1 go generate ./...
	@echo "Code generation complete!"

generate-dry-run: ## Show what would be generated without writing files
	@echo "Dry run - showing what would be generated..."
	@go run cmd/encx-gen/main.go cmd/encx-gen/config.go cmd/encx-gen/generator.go generate -dry-run .

validate: ## Validate configuration and struct tags
	@echo "Validating encx configuration and tags..."
	@go run cmd/encx-gen/main.go cmd/encx-gen/config.go cmd/encx-gen/generator.go validate -v .
	@echo "Validation complete!"

# Build targets
build: generate ## Build the project after generating code
	@echo "Building encx..."
	@go build -v ./...
	@echo "Build complete!"

build-cli: ## Build the encx-gen CLI tool
	@echo "Building encx-gen CLI..."
	@go build -o bin/encx-gen ./cmd/encx-gen
	@echo "CLI built at bin/encx-gen"

install-cli: build-cli ## Install encx-gen CLI to GOPATH/bin
	@echo "Installing encx-gen CLI..."
	@go install ./cmd/encx-gen
	@echo "encx-gen installed to $(shell go env GOPATH)/bin/encx-gen"

# Test targets
test: generate ## Run tests after generating code
	@echo "Running tests..."
	@go test -v ./...

test-race: generate ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v ./...

test-cover: generate ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -cover -v ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: generate ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Code quality targets
lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install it from https://golangci-lint.run/"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Development targets
dev-setup: install-tools ## Set up development environment
	@echo "Setting up development environment..."
	@go mod tidy
	@go mod download

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# CI/CD targets
ci: validate generate test lint vet ## Run full CI pipeline
	@echo "CI pipeline complete!"

ci-check-generated: generate ## Check if generated code is up to date (for CI)
	@echo "Checking if generated code is up to date..."
	@if git diff --exit-code; then \
		echo "✓ Generated code is up to date"; \
	else \
		echo "✗ Generated code is out of date. Please run 'make generate' and commit changes."; \
		exit 1; \
	fi

# Cleanup targets
clean: ## Clean build artifacts and generated files
	@echo "Cleaning up..."
	@rm -f bin/encx-gen
	@rm -f coverage.out coverage.html
	@rm -f .encx-gen-cache.json
	@find . -name "*_encx.go" -type f -delete
	@echo "Cleanup complete!"

clean-cache: ## Clean only the generation cache
	@echo "Cleaning generation cache..."
	@rm -f .encx-gen-cache.json
	@echo "Cache cleaned!"

# Docker targets (if using Docker)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t encx:latest .

docker-test: ## Run tests in Docker
	@echo "Running tests in Docker..."
	@docker run --rm encx:latest make test

# Release targets
version: ## Show current version
	@go run cmd/encx-gen/main.go cmd/encx-gen/config.go cmd/encx-gen/generator.go version

# Documentation targets
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go doc -all ./... > docs/api.txt
	@echo "Documentation generated in docs/api.txt"

# Example targets
example-init: build-cli ## Initialize example configuration
	@echo "Creating example configuration..."
	@./bin/encx-gen init -force
	@echo "Example configuration created in encx.yaml"

example-generate: example-init ## Run example generation
	@echo "Running example generation..."
	@./bin/encx-gen generate -v .

# Environment-specific targets
dev: fmt vet generate test ## Development workflow: format, vet, generate, test

prod-build: ci build ## Production build with full checks

# Help text for common workflows
workflows: ## Show common development workflows
	@echo "Common development workflows:"
	@echo ""
	@echo "  Development:"
	@echo "    make dev              # Format, vet, generate, test"
	@echo "    make generate         # Generate encx code"
	@echo "    make validate         # Validate configuration"
	@echo ""
	@echo "  Building:"
	@echo "    make build            # Build project with code generation"
	@echo "    make build-cli        # Build encx-gen CLI tool"
	@echo "    make install-cli      # Install CLI to GOPATH/bin"
	@echo ""
	@echo "  Testing:"
	@echo "    make test             # Run tests"
	@echo "    make test-cover       # Run tests with coverage"
	@echo "    make benchmark        # Run benchmarks"
	@echo ""
	@echo "  CI/CD:"
	@echo "    make ci               # Full CI pipeline"
	@echo "    make ci-check-generated # Check generated code is up to date"
