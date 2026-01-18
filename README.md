# Arbiter

An agentic based coding orchestrator for both on-prem and off-prem development.

## Architecture

Arbiter is built with the following principles:

- **Go-First Implementation**: All primary functionality is implemented in Go for performance, maintainability, and minimal host footprint
- **Containerized Everything**: Every component runs in containers with no exceptions, ensuring consistency across environments
- **Minimal Language Footprint**: While other languages (Python, shell scripts) can be used when more appropriate, we exercise caution to minimize the number of languages and dependencies on the host system

## Prerequisites

- Docker (20.10+)
- Docker Compose (1.29+)
- Go 1.21+ (for local development only)
- Make (optional, for convenience commands)

## Quick Start

### Running with Docker (Recommended)

```bash
# Build and run using docker-compose
make docker-run

# Or manually
docker-compose up -d

# View logs
docker-compose logs -f arbiter

# Stop the service
make docker-stop
```

### Building the Docker Image

```bash
make docker-build

# Or manually
docker build -t arbiter:latest .
```

### Local Development

For local development without Docker:

```bash
# Build the binary
make build

# Run the application
make run

# Run tests
make test

# Run linters
make lint
```

## Usage

Once running, Arbiter provides an orchestration service for coding tasks:

```bash
# Check version
docker exec arbiter /app/arbiter version

# Get help
docker exec arbiter /app/arbiter help
```

## Project Structure

```
arbiter/
├── cmd/
│   └── arbiter/          # Main application entry point
│       └── main.go
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Container orchestration
├── Makefile             # Development convenience commands
├── go.mod               # Go module definition
└── README.md            # This file
```

## Development Guidelines

1. **Primary Language**: Implement all core functionality in Go
2. **Containerization**: All services, tools, and components must run in containers
3. **Additional Languages**: Only use Python or shell scripts when they provide clear advantages, and document the rationale
4. **Security**: Run containers as non-root users, use multi-stage builds to minimize image size
5. **Testing**: All code should be tested; use Go's built-in testing framework

## Contributing

When contributing to this project:

1. Ensure all code follows the architecture principles above
2. All new features must be containerized
3. Update documentation for any new features or changes
4. Run tests and linters before submitting changes

## License

See LICENSE file for details.
