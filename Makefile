.PHONY: all build build-all run start stop restart test test-docker test-api coverage fmt vet lint lint-yaml lint-docs deps clean distclean install config dev-setup docker-build docker-up docker-down help release release-major release-minor release-patch

# Build variables
BINARY_NAME=loom
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
PIDFILE=.loom.pid

all: build

# Build the Go binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/loom

# Build for multiple platforms
build-all: lint-yaml
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/loom
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/loom
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/loom
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/loom

# Start loom locally (native, no Docker)
start: build
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		echo "Loom is already running (PID $$(cat $(PIDFILE)))"; \
	else \
		echo "Starting loom (logging to loom.log)..."; \
		bash -c './$(BINARY_NAME) > loom.log 2>&1 & echo $$! > $(PIDFILE)'; \
		sleep 2; \
		if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
			echo "Loom started (PID $$(cat $(PIDFILE)), http://localhost:8081)"; \
		else \
			echo "Loom failed to start. Check loom.log"; \
			rm -f $(PIDFILE); \
			exit 1; \
		fi; \
	fi

# Stop locally-running loom
stop:
	@if [ -f $(PIDFILE) ]; then \
		pid=$$(cat $(PIDFILE)); \
		if kill -0 $$pid 2>/dev/null; then \
			echo "Stopping loom (PID $$pid)..."; \
			kill $$pid && rm -f $(PIDFILE) && echo "Stopped"; \
		else \
			echo "Loom not running (stale pidfile)"; \
			rm -f $(PIDFILE); \
		fi; \
	else \
		echo "Loom not running (no pidfile)"; \
	fi

# Restart locally-running loom (stop + start)
restart: stop start

# Run tests locally
test:
	go test -v ./...

# Run tests in Docker (with Temporal)
test-docker:
	docker compose up -d --build temporal-postgresql temporal temporal-ui
	docker compose run --rm loom-test
	docker compose down

# Run post-flight API tests
test-api:
	@echo "Running post-flight API tests..."
	@./tests/postflight/api_test.sh $(BASE_URL)

# Run tests with coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run all linters
lint: fmt vet lint-yaml lint-docs

lint-yaml:
	go run ./cmd/yaml-lint

lint-docs:
	@bash scripts/check-docs-structure.sh

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*-* $(BINARY_NAME)-*.exe
	rm -f coverage.out coverage.html
	rm -f *.db

# Deep clean: stop containers, remove images, prune docker, clean all
distclean: clean
	@docker compose down -v --remove-orphans 2>/dev/null || true
	@docker rmi loom:latest loom-loom-test:latest 2>/dev/null || true
	@docker image prune -f
	@go clean -cache -testcache

# Install binary to $GOPATH/bin
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Create config.yaml from example if missing
config:
	@if [ ! -f config.yaml ]; then \
		cp config.yaml.example config.yaml; \
		echo "Created config.yaml from example"; \
	else \
		echo "config.yaml already exists"; \
	fi

# Development setup
dev-setup: deps config
	@echo "Development environment setup complete"
	@echo "Run 'make start' to start the server"

# Docker targets
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Run full stack in Docker (foreground)
run:
	docker compose up --build

# Release automation
release:
	@BATCH=$(BATCH) ./scripts/release.sh patch

release-minor:
	@BATCH=$(BATCH) ./scripts/release.sh minor

release-major:
	@BATCH=$(BATCH) ./scripts/release.sh major

help:
	@echo "Loom - Makefile Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the Go binary"
	@echo "  make build-all    - Cross-compile for linux/darwin/windows"
	@echo "  make start        - Build and start loom locally (native)"
	@echo "  make stop         - Stop locally-running loom"
	@echo "  make restart      - Stop and restart loom locally"
	@echo "  make test         - Run tests locally"
	@echo "  make test-docker  - Run tests in Docker (with Temporal)"
	@echo "  make test-api     - Run post-flight API tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make lint         - Run all linters (fmt, vet, yaml, docs)"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make distclean    - Deep clean (docker + build cache)"
	@echo "  make install      - Install binary to GOPATH/bin"
	@echo "  make config       - Create config.yaml from example"
	@echo "  make dev-setup    - Set up development environment"
	@echo ""
	@echo "Docker:"
	@echo "  make run          - Run full stack in Docker (foreground)"
	@echo "  make docker-build - Build Docker images"
	@echo "  make docker-up    - Start containers (background)"
	@echo "  make docker-down  - Stop containers"
	@echo ""
	@echo "Release:"
	@echo "  make release       - Patch release (x.y.Z)"
	@echo "  make release-minor - Minor release (x.Y.0)"
	@echo "  make release-major - Major release (X.0.0)"
