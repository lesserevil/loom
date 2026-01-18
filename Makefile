.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean lint

# Default target
help:
	@echo "Arbiter - Makefile Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the Go binary"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run application in Docker"
	@echo "  make docker-stop  - Stop Docker containers"
	@echo "  make docker-clean - Clean Docker resources"
	@echo ""

# Build the Go binary
build:
	@echo "Building arbiter..."
	@go build -o bin/arbiter ./cmd/arbiter

# Run the application locally
run: build
	@echo "Running arbiter..."
	@./bin/arbiter

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linters
lint:
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t arbiter:latest .

# Run application in Docker using docker-compose
docker-run:
	@echo "Starting arbiter in Docker..."
	@docker-compose up -d

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Clean Docker resources
docker-clean: docker-stop
	@echo "Cleaning Docker resources..."
	@docker-compose down -v
	@docker rmi arbiter:latest || true
