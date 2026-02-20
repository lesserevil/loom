# Per-Project Container Architecture

## Problem Statement

Currently, all projects share a single Loom container, which creates several issues:
1. **No hermetic execution**: Projects can't install their own dependencies
2. **No root access**: Can't apt-get install or system-level tools per project
3. **Shared filesystem**: File conflicts and security boundaries unclear
4. **Resource contention**: One project can starve others

## Requirements (P0)

- Each project runs in its own isolated container
- Projects can install system packages (apt, yum, etc.) as root
- Hermetic execution - project A's deps don't affect project B
- Orchestration via docker-compose or Kubernetes
- Communication between project containers and Loom control plane

## Proposed Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Loom Control Plane                       │
│  - Dispatcher, BeadsManager, ProviderRegistry               │
│  - Web UI, API Server                                        │
│  - Database (SQLite or Postgres)                             │
└──────────────────┬──────────────────────────────────────────┘
                   │
           gRPC/HTTP API
                   │
      ┌────────────┼────────────┐
      │            │            │
┌─────▼────┐  ┌───▼─────┐  ┌──▼──────┐
│ Project  │  │ Project │  │ Project │
│ Container│  │ Container│ │Container│
│  (loom)  │  │(aviation)│ │  (foo)  │
├──────────┤  ├──────────┤ ├─────────┤
│ - Clone  │  │ - Clone  │ │ - Clone │
│ - Agent  │  │ - Agent  │ │ - Agent │
│ - Exec   │  │ - Exec   │ │ - Exec  │
│ - Build  │  │ - Build  │ │ - Build │
└──────────┘  └──────────┘ └─────────┘
```

### Container Types

1. **Loom Control Plane** (1 container)
   - Orchestrates all project containers
   - Runs dispatcher, API, web UI
   - Manages beads and agent assignments
   - Stores state in database

2. **Project Containers** (N containers, one per project)
   - Based on configurable base image (default: ubuntu:22.04)
   - Has git, build tools, language runtimes
   - Runs project-specific agent worker
   - Executes actions (build, test, git commit, etc.)
   - Has root access for installing dependencies

### Communication Flow

1. **Dispatcher** (control plane) selects bead for project
2. **Control plane** calls project container's agent via gRPC/HTTP
3. **Project agent** executes task in isolated environment
4. **Project agent** reports results back to control plane
5. **Control plane** updates bead status and database

### Configuration Schema

```yaml
projects:
  - id: loom
    name: Loom
    git_repo: "git@github.com:jordanhubbard/loom.git"
    branch: main

    # Container config
    container:
      image: "ubuntu:22.04"  # Base image
      cpu_limit: "2.0"        # CPU cores
      memory_limit: "4GB"     # Memory

      # Pre-install system packages
      apt_packages:
        - build-essential
        - golang-1.22
        - git

      # Custom Dockerfile for complex setup
      dockerfile: "./docker/loom-project.Dockerfile"  # Optional

      # Environment variables
      env:
        GOPATH: /workspace/go
        PATH: /usr/local/go/bin:$PATH

      # Volume mounts
      volumes:
        - ./data/projects/loom:/workspace
        - ~/.ssh:/root/.ssh:ro
```

### Docker Compose Implementation

```yaml
# docker-compose.yml (generated dynamically)
version: '3.8'

services:
  # Control plane
  loom-control:
    image: loom:latest
    ports:
      - "8080:8080"
    environment:
      - LOOM_MODE=control-plane
    volumes:
      - loom-db:/app/data
    networks:
      - loom-network

  # Project: loom
  loom-project-loom:
    image: loom-project:loom
    environment:
      - PROJECT_ID=loom
      - CONTROL_PLANE_URL=http://loom-control:8080
    volumes:
      - loom-project-workspace:/workspace
      - ~/.ssh:/root/.ssh:ro
    networks:
      - loom-network
    cap_add:
      - SYS_ADMIN  # For hermetic operations

  # Project: aviation
  loom-project-aviation:
    image: loom-project:aviation
    environment:
      - PROJECT_ID=aviation
      - CONTROL_PLANE_URL=http://loom-control:8080
    volumes:
      - aviation-project-workspace:/workspace
    networks:
      - loom-network

networks:
  loom-network:
    driver: bridge

volumes:
  loom-db:
  loom-project-workspace:
  aviation-project-workspace:
```

### Kubernetes Implementation

```yaml
# k8s/loom-control-plane.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: loom-control-plane
spec:
  replicas: 1
  selector:
    matchLabels:
      app: loom-control
  template:
    metadata:
      labels:
        app: loom-control
    spec:
      containers:
      - name: loom
        image: loom:latest
        ports:
        - containerPort: 8080
        env:
        - name: LOOM_MODE
          value: "control-plane"
---
# k8s/loom-project-loom.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: loom-project-loom
spec:
  replicas: 1
  selector:
    matchLabels:
      app: loom-project
      project: loom
  template:
    metadata:
      labels:
        app: loom-project
        project: loom
    spec:
      containers:
      - name: project-agent
        image: loom-project:loom
        env:
        - name: PROJECT_ID
          value: "loom"
        - name: CONTROL_PLANE_URL
          value: "http://loom-control-plane:8080"
        securityContext:
          privileged: true  # For hermetic operations
```

## Implementation Plan

### Phase 1: Project Agent Service
- [ ] Create `cmd/loom-project-agent` - lightweight agent for project containers
- [ ] Implement gRPC/HTTP API for receiving tasks from control plane
- [ ] Action executor that runs in project container context
- [ ] Result reporter back to control plane

### Phase 2: Container Orchestration
- [ ] Dynamic docker-compose.yml generator
- [ ] Container lifecycle management (start, stop, restart)
- [ ] Health checks and auto-restart
- [ ] Resource limits enforcement

### Phase 3: Control Plane Modifications
- [ ] Dispatcher routes tasks to project container agents (not local workers)
- [ ] Project container registration and discovery
- [ ] gRPC client for control plane → project communication

### Phase 4: Configuration & UX
- [ ] Extended project config schema with container options
- [ ] CLI commands: `loomctl project start/stop/logs`
- [ ] Web UI showing project container status
- [ ] Migration guide for existing projects

### Phase 5: Kubernetes Support
- [ ] K8s manifests generator
- [ ] Helm chart for deployment
- [ ] Service mesh integration (optional)

## Migration Strategy

### For Existing Loom Installations

1. **Backward Compatibility**: Keep single-container mode as default
2. **Opt-in per project**: `use_container: true` in project config
3. **Gradual migration**: Convert one project at a time
4. **Testing**: Run both modes in parallel during transition

### Example Migration

```yaml
# Before (single container)
projects:
  - id: loom
    git_repo: "..."
    branch: main

# After (per-project containers)
projects:
  - id: loom
    git_repo: "..."
    branch: main
    use_container: true  # Enable per-project container
    container:
      image: "golang:1.22"
      apt_packages: [git, build-essential]
```

## Security Considerations

1. **Container Escape**: Use AppArmor/SELinux profiles
2. **Resource Limits**: Enforce CPU/memory quotas
3. **Network Isolation**: Projects can't directly communicate
4. **Secrets Management**: Mount secrets per-project, not shared
5. **Git Credentials**: Per-project SSH keys, not shared

## Performance Considerations

- **Startup time**: ~5-10s per container (acceptable for long-running projects)
- **Resource overhead**: ~100MB per container (minimal)
- **Network latency**: gRPC adds ~1-5ms vs local calls (negligible)
- **Scaling**: Can run 10-50 projects on typical dev machine

## Alternatives Considered

1. **Virtual Machines**: Too heavy (GB per VM vs. MB per container)
2. **Docker-in-Docker**: Complex, security issues
3. **Namespace isolation**: Not hermetic enough (can't apt-get)
4. **Chroot jails**: Too limited, no process isolation

## Success Criteria

- ✅ Projects can `apt-get install` their dependencies
- ✅ Project A's build doesn't affect project B
- ✅ Can run 10+ projects concurrently without conflicts
- ✅ Migration path preserves existing beads and history
- ✅ Performance within 10% of single-container mode

## Timeline

- **Phase 1**: 2-3 days (project agent service)
- **Phase 2**: 2-3 days (docker-compose orchestration)
- **Phase 3**: 1-2 days (control plane mods)
- **Phase 4**: 1-2 days (config & UX)
- **Phase 5**: 2-3 days (K8s support)

**Total: ~10-15 days** for complete implementation

## Next Steps

1. Review and approve this design
2. Create P0 bead for implementation
3. Start with Phase 1 (project agent service)
4. Iterate with user feedback
