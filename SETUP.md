# Loom Setup Guide

Everything you need to get Loom running and create your first project.

## Prerequisites

- Docker (20.10+)
- Docker Compose (1.29+)
- Go 1.25+ (for local development only)
- Make (optional, for convenience commands)

## Running with Docker (Recommended)

The Docker setup includes:
- Loom application server (port 8080)
- Temporal server (port 7233)
- Temporal UI (port 8088)
- PostgreSQL database for Temporal

```bash
# Build and run all services
docker compose up -d

# Verify everything is healthy
docker compose ps

# View logs
docker compose logs -f loom

# Stop all services
docker compose down
```

### Using Make Commands

```bash
# Build and run
make docker-run

# Build Docker image
make docker-build

# Stop services
make docker-stop

# Clean Docker resources
make docker-clean
```

## Connecting to the UI

Once the services are running:

- **Loom Web UI**: http://localhost:8080
- **Temporal UI**: http://localhost:8088 — view workflow executions, inspect history, monitor active workflows, debug failures

## Configuration

Configuration is managed via `config.yaml`:

```yaml
server:
  http_port: 8080
  enable_http: true

temporal:
  host: localhost:7233              # Temporal server address
  namespace: loom-default           # Temporal namespace
  task_queue: loom-tasks            # Task queue name
  workflow_execution_timeout: 24h   # Max workflow duration
  workflow_task_timeout: 10s        # Workflow task timeout
  enable_event_bus: true            # Enable event bus
  event_buffer_size: 1000           # Event buffer size

agents:
  max_concurrent: 10
  default_persona_path: ./personas
  heartbeat_interval: 30s
  file_lock_timeout: 10m
```

Create your config from the example:
```bash
cp config.yaml.example config.yaml
# or
make config
```

## Bootstrapping Your First Project

Projects are registered via `config.yaml` under `projects:` (and persisted in the configuration DB when enabled).

Required fields:
- `id`, `name`, `git_repo`, `branch`, `beads_path`

Optional fields:
- `is_perpetual` (never closes)
- `context` (recommended: build/test/lint commands and other agent-relevant context)

Example:

```yaml
projects:
  - id: loom
    name: Loom
    git_repo: https://github.com/jordanhubbard/loom
    branch: main
    beads_path: .beads
    is_perpetual: true
    context:
      test: go test ./...
      vet: go vet ./...
```

Loom "dogfoods" itself by registering this repo as a project and loading beads from the project's `.beads/` directory.

## Local Development

### Building Locally

```bash
# Install dependencies
go mod download

# Build the binary
go build -o loom ./cmd/loom

# Run the application
./loom
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/temporal/...
```

### Development with Temporal

For local development with Temporal:

1. Start Temporal server:
```bash
docker compose up -d temporal temporal-postgresql temporal-ui
```

2. Build and run loom locally:
```bash
go build -o loom ./cmd/loom
./loom
```

3. Access Temporal UI:
```bash
open http://localhost:8088
```

## Monitoring

### Event Stream

Monitor real-time events:
```bash
# Watch all events
curl -N http://localhost:8080/api/v1/events/stream

# Monitor specific project
curl -N "http://localhost:8080/api/v1/events/stream?project_id=my-project"
```

### Logs

View service logs:
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f loom
docker compose logs -f temporal
```

## Troubleshooting

### Temporal Connection Issues

If loom can't connect to Temporal:

1. Check Temporal is running:
```bash
docker compose ps temporal
```

2. Check Temporal logs:
```bash
docker compose logs temporal
```

3. Verify connectivity:
```bash
docker exec loom nc -zv temporal 7233
```

### Workflow Not Starting

If workflows aren't starting:

1. Check worker is running:
```bash
docker compose logs loom | grep "Temporal worker"
```

2. Verify task queue in Temporal UI
3. Check workflow registration in logs

### Event Stream Not Working

If event stream endpoint returns errors:

1. Verify Temporal is enabled in config
2. Check event bus initialization:
```bash
docker compose logs loom | grep "event bus"
```
