# AgentiCorp Architecture

## Overview

AgentiCorp is a secure AI agent orchestration system that manages hierarchical teams of AI agents working on projects. It provides a complete framework for organizing, coordinating, and monitoring AI agents with different roles and capabilities.

## Conceptual Model

AgentiCorp follows a hierarchical organizational model:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        AgentiCorp (Global State)                        │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Global Providers                             │   │
│  │  (OpenAI, Anthropic, vLLM, local models - shared across all)    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        Projects                                  │   │
│  │  ┌───────────────────────────────────────────────────────────┐  │   │
│  │  │  Project A (can have sub-projects via parent_id)          │  │   │
│  │  │  ┌─────────────────────────────────────────────────────┐  │  │   │
│  │  │  │              Org Chart                              │  │  │   │
│  │  │  │  ┌─────────────────────────────────────────────┐    │  │  │   │
│  │  │  │  │  Position: CEO ──► Agent Instance           │    │  │  │   │
│  │  │  │  │  Position: CFO ──► Agent Instance           │    │  │  │   │
│  │  │  │  │  Position: PM  ──► Agent Instance           │    │  │  │   │
│  │  │  │  │  Position: EM  ──► Agent Instance           │    │  │  │   │
│  │  │  │  │  ...                                        │    │  │  │   │
│  │  │  │  └─────────────────────────────────────────────┘    │  │  │   │
│  │  │  └─────────────────────────────────────────────────────┘  │  │   │
│  │  └───────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Key Concepts

| Concept | Scope | Description |
|---------|-------|-------------|
| **Provider** | Global | AI backend (OpenAI, Anthropic, vLLM). Shared across all projects. |
| **Project** | Global | Work context with git repo, beads, and team. Can have sub-projects. |
| **Org Chart** | Per-Project | Defines team structure and role hierarchy. |
| **Position** | Per-Org Chart | A "slot" for a role (CEO, PM, Engineer). Has persona and reports-to. |
| **Agent** | Per-Project | Instance of an AI filling a position. Uses a provider for inference. |
| **Persona** | Template | Role definition (mission, character, tone) loaded from filesystem. |

### Hierarchy

```
Provider (global)
    └── Used by Agents across all projects

Project
    ├── parent_id → Project (sub-project support)
    ├── Org Chart
    │       ├── Position (CEO)
    │       │       └── Agent Instance ──uses──► Provider
    │       ├── Position (CFO)
    │       │       └── Agent Instance ──uses──► Provider
    │       └── Position (PM) reports_to CEO
    │               └── Agent Instance ──uses──► Provider
    └── Beads (work items)
```

## Core Concepts

### Providers

A **Provider** is an AI inference backend. Providers are global resources shared across all projects.

- **Types**: OpenAI, Anthropic, vLLM, local models
- **Credentials**: API keys stored encrypted in Key Manager
- **Health**: Monitored via heartbeats (latency, availability)
- **Selection**: Best provider chosen based on quality score and latency

### Projects

A **Project** represents a body of work with its own git repository, work items (beads), and team.

- **Hierarchy**: Projects can have sub-projects via `parent_id`
- **Lifecycle**: open → closed (can be reopened)
- **Perpetual**: Some projects (like AgentiCorp itself) never close
- **Context**: Build/test/lint commands for agent awareness

### Org Charts

An **Org Chart** defines the team structure for a project. Each project has one org chart that specifies which positions exist and how they relate.

- **Template**: Default org chart cloned for new projects
- **Positions**: Role slots that agents fill
- **Hierarchy**: Positions can have `reports_to` relationships

### Positions

A **Position** is a role slot within an org chart. It defines:

- **Role Name**: e.g., "ceo", "product-manager", "qa-engineer"
- **Persona Path**: Template for the role's behavior
- **Required**: Whether the position must be filled
- **Max Instances**: How many agents can fill this role (0 = unlimited)
- **Reports To**: Hierarchical relationship to another position

### Agents

An **Agent** is an AI instance that fills a position and performs work.

- **Status**: paused, idle, working, blocked
- **Paused**: Created without a provider (awaiting assignment)
- **Provider**: The AI backend used for inference
- **Current Bead**: The work item currently assigned

### Personas

A **Persona** is a template that defines an agent's behavior:

- **Mission**: What the agent is trying to accomplish
- **Character**: Personality traits and approach
- **Tone**: Communication style
- **Autonomy Level**: How much independence the agent has

Default personas are stored in `personas/default/`.

## Database Schema

### Entity Relationship Diagram

```
┌─────────────────┐
│   providers     │  (global)
│─────────────────│
│ id              │
│ name            │
│ type            │
│ endpoint        │
│ status          │
│ key_id          │
└────────┬────────┘
         │
         │ provider_id (optional)
         ▼
┌─────────────────┐       ┌─────────────────┐
│    projects     │       │   org_charts    │
│─────────────────│       │─────────────────│
│ id              │◄──────│ project_id      │
│ name            │       │ id              │
│ git_repo        │       │ name            │
│ parent_id ──────┼───┐   │ is_template     │
│ status          │   │   └────────┬────────┘
│ closed_at       │   │            │
└─────────────────┘   │            │ org_chart_id
         ▲            │            ▼
         └────────────┘   ┌─────────────────┐
                          │ org_chart_      │
                          │ positions       │
                          │─────────────────│
                          │ id              │
                          │ role_name       │
                          │ persona_path    │
                          │ reports_to ─────┼───┐
                          │ required        │   │
                          │ max_instances   │   │
                          └────────┬────────┘   │
                                   │            │
                                   └────────────┘
                                   │
                                   │ position_id
                                   ▼
                          ┌─────────────────┐
                          │    agents       │
                          │─────────────────│
                          │ id              │
                          │ name            │
                          │ role            │
                          │ project_id      │
                          │ provider_id ────┼──► providers.id
                          │ status          │
                          │ current_bead    │
                          └─────────────────┘
```

### Tables

#### providers (Global)
```sql
CREATE TABLE providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,              -- openai, anthropic, vllm, local
    endpoint TEXT NOT NULL,
    model TEXT,
    requires_key BOOLEAN NOT NULL,
    key_id TEXT,                     -- Reference to Key Manager
    status TEXT NOT NULL,            -- active, inactive, error
    last_heartbeat_at DATETIME,
    last_heartbeat_latency_ms INTEGER,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

#### projects (Hierarchical)
```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    git_repo TEXT NOT NULL,
    branch TEXT NOT NULL,
    beads_path TEXT NOT NULL,
    parent_id TEXT,                  -- Sub-project support
    is_perpetual BOOLEAN NOT NULL,
    status TEXT NOT NULL,            -- open, closed
    closed_at DATETIME,
    context_json TEXT,               -- Build/test commands
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES projects(id) ON DELETE SET NULL
);
```

#### org_charts (Per-Project)
```sql
CREATE TABLE org_charts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    is_template BOOLEAN NOT NULL,
    parent_id TEXT,                  -- Inherit from another org chart
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES org_charts(id) ON DELETE SET NULL
);
```

#### org_chart_positions (Role Slots)
```sql
CREATE TABLE org_chart_positions (
    id TEXT PRIMARY KEY,
    org_chart_id TEXT NOT NULL,
    role_name TEXT NOT NULL,
    persona_path TEXT NOT NULL,
    required BOOLEAN NOT NULL,
    max_instances INTEGER NOT NULL,  -- 0 = unlimited
    reports_to TEXT,                 -- Hierarchical relationship
    created_at DATETIME NOT NULL,
    FOREIGN KEY (org_chart_id) REFERENCES org_charts(id) ON DELETE CASCADE,
    FOREIGN KEY (reports_to) REFERENCES org_chart_positions(id) ON DELETE SET NULL
);
```

#### agents (Per-Project Instances)
```sql
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    role TEXT,
    persona_name TEXT,
    provider_id TEXT,                -- NULL = paused (no provider)
    status TEXT NOT NULL,            -- paused, idle, working, blocked
    current_bead TEXT,
    project_id TEXT,
    position_id TEXT,                -- Link to org chart position
    started_at DATETIME NOT NULL,
    last_active DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
    FOREIGN KEY (position_id) REFERENCES org_chart_positions(id) ON DELETE SET NULL,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
);
```

## Architecture Diagram

```mermaid
flowchart LR
  subgraph UI[Web UI]
    Browser[Browser]
  end

  subgraph AgentiCorp[AgentiCorp (Go)]
    API[HTTP API]
    SSE[SSE /api/v1/events/stream]
    EB[Event Bus]
    DISP[Dispatcher]
    WM[WorkerManager]
    OCM[OrgChartManager]
    PR[Provider Registry]
    HB[Provider Heartbeats]
    KM[Key Manager]
    CFG[Config DB (SQLite)]
    BM[Beads Manager]
    PM[Project Manager]
    DM[Decision Manager]
  end

  subgraph Temporal[Temporal (optional)]
    TW[Temporal Worker]
    TS[Temporal Server]
  end

  subgraph Providers[Model Providers]
    VLLM[vLLM / OpenAI-compatible]
    OAI[OpenAI/Anthropic/etc]
  end

  Browser --> API
  API -->|events| EB
  EB --> SSE

  API --> CFG
  API --> PM
  API --> OCM
  API --> BM
  API --> DM
  API --> WM
  API --> PR
  API --> KM

  OCM --> WM
  DISP --> BM
  DISP --> WM
  WM --> PR
  PR --> HB
  PR --> VLLM
  PR --> OAI

  EB -. signal .-> TW
  TW --> TS
```

## Default Org Chart

When a new project is created, it receives a default org chart with these positions:

| Position | Role | Reports To | Required |
|----------|------|------------|----------|
| CEO | ceo | - | Yes |
| CFO | cfo | CEO | No |
| Product Manager | product-manager | CEO | Yes |
| Project Manager | project-manager | CEO | No |
| Engineering Manager | engineering-manager | CEO | Yes |
| Code Reviewer | code-reviewer | EM | No |
| QA Engineer | qa-engineer | EM | No |
| DevOps Engineer | devops-engineer | EM | No |
| Documentation Manager | documentation-manager | PM | No |
| Web Designer | web-designer | PM | No |
| Web Designer Engineer | web-designer-engineer | PM | No |
| Public Relations Manager | public-relations-manager | CEO | No |
| Decision Maker | decision-maker | CEO | No |
| Housekeeping Bot | housekeeping-bot | - | No |

## Agent Lifecycle

```
                    ┌──────────┐
    Create Agent    │  paused  │  No provider assigned
    (org chart      └────┬─────┘
     bootstrap)          │
                         │ Assign Provider
                         ▼
                    ┌──────────┐
                    │   idle   │  Ready for work
                    └────┬─────┘
                         │
              ┌──────────┴──────────┐
              │ Assign Bead         │ Blocked (waiting)
              ▼                     ▼
         ┌──────────┐         ┌──────────┐
         │ working  │────────►│ blocked  │
         └────┬─────┘         └────┬─────┘
              │                    │
              │ Complete Bead      │ Unblocked
              ▼                    ▼
         ┌──────────┐         ┌──────────┐
         │   idle   │◄────────│ working  │
         └────┬─────┘         └──────────┘
              │
              │ Stop Agent
              ▼
         ┌──────────┐
         │ shutdown │
         └──────────┘
```

## Entity Versioning System

AgentiCorp uses a robust entity versioning system to handle schema evolution without breaking backward compatibility.

### Core Concepts

Every entity (Agent, Project, Provider, OrgChart, Position, Persona, Bead) includes:

1. **SchemaVersion**: Tracks the schema version (e.g., "1.0", "1.1")
2. **Attributes**: Extensible `map[string]any` for adding fields without schema changes
3. **MigrationRegistry**: Handles version-to-version transformations

### When to Use Each Approach

| Change Type | Approach | Requires Migration |
|-------------|----------|-------------------|
| Add optional field | Use `Attributes` map | No |
| Add required field | Bump schema version | Yes |
| Rename field | Bump schema version | Yes (breaking) |
| Delete field | Move to Attributes, bump version | Yes |
| Change field type | Bump schema version | Yes (breaking) |
| Add behavior | Use feature flag in Attributes | No |

### Attributes

The `Attributes` map provides extensible storage for optional fields:

```go
// Set an attribute
agent.SetAttribute("ui.color", "#FF5733")
agent.SetAttribute("metrics.run_count", 42)

// Get typed attributes with defaults
color := agent.GetStringAttribute("ui.color", "#000000")
count := agent.GetIntAttribute("metrics.run_count", 0)
enabled := agent.GetBoolAttribute("feature.beta", false)
```

Standard attribute namespaces:
- `ui.*` - UI display hints (color, icon, display_name)
- `metrics.*` - Runtime statistics
- `feature.*` - Feature flags
- `behavior.*` - Behavioral configuration
- `legacy.*` - Deprecated fields preserved for compatibility

### Migration Registry

Migrations transform entities from one schema version to another:

```go
// Register a migration
RegisterMigration(
    EntityTypeAgent, "1.0", "1.1",
    "Add default tags and rename role to job_title",
    true, // Breaking change
    func(entity VersionedEntity) error {
        agent := entity.(*Agent)
        // Preserve old value
        agent.SetAttribute("legacy.role", agent.Role)
        // Set defaults for new fields
        if !agent.HasAttribute("tags") {
            agent.SetAttribute("tags", []string{})
        }
        return nil
    },
)
```

### Migration on Load

Entities are automatically migrated when loaded from the database:

```go
// Ensure entity is at latest version
if err := EnsureMigrated(agent); err != nil {
    log.Printf("Migration failed: %v", err)
}
```

### Database Schema

All entity tables include versioning columns:

```sql
-- Added to every entity table
schema_version TEXT NOT NULL DEFAULT '1.0',
attributes_json TEXT
```

### Best Practices

1. **Never remove fields** - Move to Attributes with `legacy.*` prefix
2. **Use Attributes for optional data** - Avoids schema changes
3. **Mark breaking migrations** - Set `breaking: true` for field renames/deletes
4. **Test migrations** - Write tests for each migration path
5. **Document changes** - Add description to each migration

## Security Model

### Key Manager

The Key Manager securely stores provider API credentials:

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Derivation**: PBKDF2 with SHA-256 (100,000 iterations)
- **Salt**: 32 bytes per key (unique)
- **Nonce**: 12 bytes per key (unique)

### Password Handling

- **Never stored**: Password exists only in memory
- **Environment variable**: `AGENTICORP_PASSWORD` for automation
- **Interactive prompt**: Hidden input for security
- **Memory clearing**: Password cleared when key manager locks

## Temporal Workflow Integration

AgentiCorp uses [Temporal](https://temporal.io) for durable workflow orchestration:

### Workflows

| Workflow | Purpose |
|----------|---------|
| Agent Lifecycle | Manages agent state (spawned → working → idle → shutdown) |
| Bead Processing | Tracks work item lifecycle (open → in_progress → closed) |
| Decision | Handles approval workflows with timeout |
| Event Aggregator | Collects and distributes events |
| Provider Heartbeat | Monitors provider health |

### Event Bus

Real-time event streaming via SSE:

```
Event Types:
- agent.spawned        - New agent created
- agent.status_change  - Agent status updated
- bead.created         - New work item created
- bead.assigned        - Work assigned to agent
- decision.created     - Decision point created
- decision.resolved    - Decision made
```

## Directory Structure

```
agenticorp/
├── cmd/agenticorp/          # Main application entry point
├── internal/
│   ├── agent/               # Agent management
│   ├── agenticorp/          # Core orchestrator
│   ├── api/                 # HTTP API handlers
│   ├── beads/               # Work item management
│   ├── database/            # SQLite database layer
│   ├── decision/            # Decision framework
│   ├── keymanager/          # Credential encryption
│   ├── orgchart/            # Org chart management
│   ├── project/             # Project management
│   ├── provider/            # Provider registry
│   └── temporal/            # Temporal integration
├── pkg/
│   ├── config/              # Configuration
│   └── models/              # Shared data models
├── personas/
│   ├── agenticorp/          # Self-improvement persona
│   └── default/             # Default role personas
├── web/
│   └── static/              # Web UI (HTML, CSS, JS)
├── config.yaml              # Configuration file
├── docker-compose.yml       # Container orchestration
└── Dockerfile               # Multi-stage Docker build
```

## API Overview

### Projects
- `GET /api/v1/projects` - List all projects
- `GET /api/v1/projects/{id}` - Get project details
- `POST /api/v1/projects/{id}/close` - Close project
- `POST /api/v1/projects/{id}/reopen` - Reopen project

### Agents
- `GET /api/v1/agents` - List all agents
- `POST /api/v1/agents` - Spawn new agent
- `GET /api/v1/agents/{id}` - Get agent details
- `PUT /api/v1/agents/{id}` - Update agent
- `DELETE /api/v1/agents/{id}` - Stop agent
- `POST /api/v1/agents/{id}/clone` - Clone agent with new persona

### Providers
- `GET /api/v1/providers` - List all providers
- `POST /api/v1/providers` - Create provider
- `GET /api/v1/providers/{id}` - Get provider details
- `PUT /api/v1/providers/{id}` - Update provider
- `DELETE /api/v1/providers/{id}` - Delete provider

### Events
- `GET /api/v1/events/stream` - SSE event stream
- `GET /api/v1/events/stats` - Event statistics

### Work Items
- `GET /api/v1/beads` - List beads
- `POST /api/v1/beads` - Create bead
- `GET /api/v1/beads/{id}` - Get bead
- `PUT /api/v1/beads/{id}` - Update bead

## Configuration

Configuration is managed via `config.yaml`:

```yaml
server:
  http_port: 8080

temporal:
  host: localhost:7233
  namespace: agenticorp-default
  task_queue: agenticorp-tasks

agents:
  max_concurrent: 20
  default_persona_path: ./personas
  heartbeat_interval: 30s

projects:
  - id: agenticorp
    name: AgentiCorp
    git_repo: https://github.com/jordanhubbard/agenticorp
    branch: main
    beads_path: .beads
    is_perpetual: true
```

## Future Enhancements

- [ ] Sub-project inheritance of org charts
- [ ] Org chart visual editor in UI
- [ ] Position-based access control
- [ ] Multi-tenancy support
- [ ] Agent performance metrics
- [ ] Automated provider selection based on task type
