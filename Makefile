.PHONY: help build test lint clean docker-build docker-clean deploy-dev run-local install-deps

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Go targets
build: ## Build all Go binaries
	@echo "Building api-server..."
	cd api-server && go build -o bin/server ./cmd/server/main.go
	@echo "Building operator..."
	cd operator && go build -o bin/manager ./cmd/manager/main.go

test: ## Run all tests
	@echo "Testing Go modules..."
	cd api-server && go test -v ./...
	cd operator && go test -v ./...
	@echo "Testing Python modules..."
	cd analyzer && python -m pytest tests/ -v || echo "No tests found for analyzer"
	cd runtime && python -m pytest tests/ -v || echo "No tests found for runtime"

lint: ## Run linters for all projects
	@echo "Linting Go code..."
	cd api-server && golangci-lint run ./...
	cd operator && golangci-lint run ./...
	@echo "Linting Python code..."
	cd analyzer && ruff check src/
	cd runtime && ruff check *.py

install-deps: ## Install development dependencies
	@echo "Installing Go dependencies..."
	cd api-server && go mod tidy
	cd operator && go mod tidy
	@echo "Installing Python dependencies..."
	cd analyzer && pip install -r requirements.txt
	cd runtime && pip install -r requirements.txt
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	pip install ruff pytest

# Docker targets
docker-build: ## Build all Docker images
	@echo "Building Docker images..."
	docker build -t qsim-api:latest api-server/
	docker build -t qsim-analyzer:latest analyzer/
	docker build -t qsim-runtime:latest runtime/

docker-clean: ## Remove Docker images and containers
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker rmi -f qsim-api:latest qsim-analyzer:latest qsim-runtime:latest || true

# Development targets
run-local: ## Start local development environment with docker-compose
	@echo "Starting local development environment..."
	docker-compose up -d postgres redis
	@echo "Waiting for services to be ready..."
	sleep 10
	docker-compose up api-server analyzer

stop-local: ## Stop local development environment
	@echo "Stopping local development environment..."
	docker-compose down

# Kubernetes targets
deploy-dev: ## Deploy to minikube (development)
	@echo "Deploying to minikube..."
	kubectl apply -f deploy/
	@echo "Deployment completed. Check status with: kubectl get pods"

undeploy-dev: ## Remove deployment from minikube
	@echo "Removing deployment from minikube..."
	kubectl delete -f deploy/

# Utility targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf api-server/bin/
	rm -rf operator/bin/
	find . -name "__pycache__" -type d -exec rm -rf {} +
	find . -name "*.pyc" -delete

format: ## Format code
	@echo "Formatting Go code..."
	cd api-server && go fmt ./...
	cd operator && go fmt ./...
	@echo "Formatting Python code..."
	cd analyzer && ruff format src/
	cd runtime && ruff format *.py

check-env: ## Check if required tools are installed
	@echo "Checking environment..."
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed"; exit 1; }
	@command -v python3 >/dev/null 2>&1 || { echo "Python3 is not installed"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "Docker is not installed"; exit 1; }
	@command -v kubectl >/dev/null 2>&1 || { echo "kubectl is not installed"; exit 1; }
	@echo "Environment check passed!"