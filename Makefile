# Makefile for Rancher Questions Generator
# Based on Code Review Agent specifications

.PHONY: help build test test-unit test-integration test-security test-regression test-coverage clean dev-setup dev-test docker-build docker-push

# Variables
PROJECT_NAME := rancher-questions-generator
VERSION := $(shell grep '^version:' charts/$(PROJECT_NAME)/Chart.yaml | cut -d' ' -f2)
REGISTRY := ghcr.io/wiredquill
BACKEND_DIR := backend
FRONTEND_DIR := frontend-simple
TEST_COVERAGE_THRESHOLD := 80

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development setup
dev-setup: ## Set up development environment
	@echo "🔧 Setting up development environment..."
	cd $(BACKEND_DIR) && go mod tidy
	cd $(BACKEND_DIR) && go mod download
	@echo "✅ Development environment ready"

# Build targets
build: ## Build the Go backend
	@echo "🏗️ Building backend..."
	cd $(BACKEND_DIR) && go build -o main cmd/main.go
	@echo "✅ Backend built successfully"

# Test targets
test: test-unit test-integration test-security ## Run all tests
	@echo "✅ All tests completed"

test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	cd $(BACKEND_DIR) && go test -v ./pkg/... ./internal/...
	@echo "✅ Unit tests passed"

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	cd $(BACKEND_DIR) && go test -v -tags=integration ./tests/integration/...
	@echo "✅ Integration tests passed"

test-security: ## Run security tests
	@echo "🔒 Running security tests..."
	cd $(BACKEND_DIR) && go test -v -tags=security ./tests/security/...
	@echo "✅ Security tests passed"

test-regression: ## Run regression tests
	@echo "🔄 Running regression tests..."
	cd $(BACKEND_DIR) && go test -v -tags=regression ./tests/regression/...
	@echo "✅ Regression tests passed"

test-coverage: ## Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	cd $(BACKEND_DIR) && go test -v -coverprofile=coverage.out ./pkg/... ./internal/...
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html
	cd $(BACKEND_DIR) && go tool cover -func=coverage.out
	@echo "📈 Coverage report generated: $(BACKEND_DIR)/coverage.html"

# Development testing
dev-test: ## Quick test for development
	@echo "⚡ Running quick development tests..."
	cd $(BACKEND_DIR) && go test -short -v ./pkg/... ./internal/...
	@echo "✅ Quick tests passed"

# Linting and quality checks
lint: ## Run linting and code quality checks
	@echo "🔍 Running linting..."
	cd $(BACKEND_DIR) && go vet ./...
	cd $(BACKEND_DIR) && gofmt -l .
	cd $(BACKEND_DIR) && go mod verify
	@echo "✅ Linting completed"

# Security scanning
security-scan: ## Run security vulnerability scan
	@echo "🛡️ Running security scan..."
	cd $(BACKEND_DIR) && govulncheck ./...
	@echo "✅ Security scan completed"

# Chart testing
chart-test: ## Test Helm chart
	@echo "⚓ Testing Helm chart..."
	helm lint charts/$(PROJECT_NAME)
	helm template test-release charts/$(PROJECT_NAME) --debug --dry-run
	@echo "✅ Chart tests passed"

# Docker build targets
docker-build: ## Build Docker images
	@echo "🐳 Building Docker images..."
	./scripts/build-optimized.sh $(VERSION)
	@echo "✅ Docker images built"

docker-push: ## Push Docker images to registry
	@echo "📤 Pushing Docker images..."
	docker push $(REGISTRY)/$(PROJECT_NAME)-backend:$(VERSION)
	docker push $(REGISTRY)/$(PROJECT_NAME)-frontend:$(VERSION)
	docker push $(REGISTRY)/$(PROJECT_NAME)-backend:latest
	docker push $(REGISTRY)/$(PROJECT_NAME)-frontend:latest
	@echo "✅ Docker images pushed"

# Validation targets
validate-all: lint test chart-test security-scan ## Run all validation checks
	@echo "✅ All validation checks passed"

# Benchmark targets
benchmark: ## Run performance benchmarks
	@echo "🏃 Running benchmarks..."
	cd $(BACKEND_DIR) && go test -bench=. -benchmem ./pkg/...
	@echo "✅ Benchmarks completed"

# Clean targets
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	cd $(BACKEND_DIR) && rm -f main coverage.out coverage.html
	cd $(BACKEND_DIR) && go clean -cache -testcache
	@echo "✅ Clean completed"

clean-docker: ## Clean Docker images
	@echo "🧹 Cleaning Docker images..."
	docker rmi $(REGISTRY)/$(PROJECT_NAME)-backend:$(VERSION) || true
	docker rmi $(REGISTRY)/$(PROJECT_NAME)-frontend:$(VERSION) || true
	docker system prune -f
	@echo "✅ Docker clean completed"

# Release targets
release: validate-all docker-build docker-push ## Build and release everything
	@echo "🚀 Release $(VERSION) completed successfully!"

# Development workflow
dev: clean dev-setup dev-test build ## Full development workflow
	@echo "💻 Development workflow completed"

# CI/CD targets
ci: lint test-unit test-integration chart-test ## CI pipeline tasks
	@echo "🔄 CI pipeline completed"

cd: docker-build docker-push ## CD pipeline tasks
	@echo "🚀 CD pipeline completed"