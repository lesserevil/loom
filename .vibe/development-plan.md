# Development Plan: Microservices Architecture Implementation

*Generated on 2026-02-18 by Vibe Feature MCP*
*Workflow: [epcc](https://mrsimpson.github.io/responsible-vibe-mcp/workflows/epcc)*

## Goal
Implement a true microservices architecture for Loom with message bus, service-to-service communication, and distributed database access. This will fix architectural gaps identified in MICROSERVICES_ARCHITECTURE_REVIEW.md and improve system robustness and scalability.

## Explore
<!-- beads-phase-id: explore -->

### Objectives
Understand the current architecture and identify all components that need to be modified for the microservices redesign.

### Findings

**Current State:**
- Monolithic application with partial containerization
- Per-project agent containers but no inter-service communication
- In-memory only message bus (AgentMessageBus)
- SQLite database with no service abstraction
- No service registry or discovery
- No message bus for cross-container communication

**Architecture Review Reference:**
See `docs/MICROSERVICES_ARCHITECTURE_REVIEW.md` for comprehensive analysis of gaps and solutions.

**Critical Gaps Identified:**
1. **No External Message Bus**: In-memory EventBus doesn't support cross-container communication
2. **No Service-to-Service Protocol**: No gRPC or typed contracts between services
3. **Monolithic Database**: SQLite doesn't support concurrent writes from multiple containers
4. **Isolated Project Containers**: No way for project agents to communicate with control plane

**Technology Decisions:**
- **Message Bus**: NATS with JetStream (lighter than RabbitMQ, excellent Go support)
- **Database**: PostgreSQL 15 with gRPC service for abstraction
- **Service Mesh**: Linkerd (simpler than Istio, lower overhead)
- **Protocols**: gRPC with protobuf for typed service contracts

---

## Plan
<!-- beads-phase-id: plan -->

### Phase Entrance Criteria:
- [x] Current architecture gaps are documented
- [x] Technology choices are made (NATS, PostgreSQL, gRPC, Linkerd)
- [x] Architecture review document exists
- [x] Implementation phases are defined
- [x] Success criteria are clear

### Implementation Phases

#### **Phase 1: Message Bus Foundation (Current Focus)**
**Goal**: Add NATS message bus container and implement publish/subscribe in Go

**Tasks:**
1. Add NATS container to docker-compose.yml
2. Create `internal/messagebus/nats.go` package
3. Define message schemas in `pkg/messages/`
4. Implement NatsMessageBus with publish/subscribe methods
5. Add message bus health checks
6. Update Loom initialization to connect to NATS

**Files to Create:**
- `internal/messagebus/nats.go` - NATS client wrapper
- `pkg/messages/tasks.go` - Task message schemas
- `pkg/messages/results.go` - Result message schemas
- `pkg/messages/events.go` - Event message schemas

**Files to Modify:**
- `docker-compose.yml` - Add NATS service
- `internal/loom/loom.go` - Initialize NATS connection
- `go.mod` - Add NATS dependencies

**Deliverable**: All containers can publish/subscribe to NATS topics

#### **Phase 2: Database Service (Follow-up)**
**Goal**: Replace SQLite with PostgreSQL and create gRPC database service

**Tasks:**
1. Add PostgreSQL container to docker-compose.yml
2. Create protobuf schemas for database operations
3. Implement gRPC Database Service
4. Migrate control plane to use Database Service
5. Add connection pooling (PgBouncer)

**Deliverable**: Database access via gRPC service

#### **Phase 3: Project Agent Communication (Follow-up)**
**Goal**: Enable project agents to communicate via message bus

**Tasks:**
1. Add NATS client to project-agent containers
2. Implement task subscription in project agents
3. Implement result publishing from agents
4. Update dispatcher to publish tasks instead of direct calls
5. Add correlation IDs for request tracking

**Deliverable**: Project agents receive tasks and publish results via NATS

#### **Phase 4: Connectors Service (Follow-up)**
**Goal**: Extract connector management to independent service

**Tasks:**
1. Define protobuf for connector operations
2. Implement gRPC Connectors Service
3. Update control plane to call Connectors Service
4. Update agents to call Connectors Service

**Deliverable**: Connectors as independent microservice

#### **Phase 5: Service Mesh & Observability (Follow-up)**
**Goal**: Add service mesh for security and observability

**Tasks:**
1. Add Linkerd service mesh
2. Configure mTLS between services
3. Add distributed tracing (Jaeger)
4. Add centralized logging (Loki)

**Deliverable**: Full observability and security

---

## Code
<!-- beads-phase-id: code -->

### Phase Entrance Criteria:
- [ ] Plan is approved and understood
- [ ] Phase 1 scope is clear (NATS message bus foundation)
- [ ] Technology stack is confirmed (NATS with JetStream)
- [ ] Message schemas are designed
- [ ] Integration points are identified

### Implementation Notes
*Code implementation details will be documented here as work progresses*

---

## Commit
<!-- beads-phase-id: commit -->

### Phase Entrance Criteria:
- [ ] All Phase 1 code is implemented
- [ ] Code compiles without errors
- [ ] NATS container starts successfully
- [ ] Message publishing works
- [ ] Message subscription works
- [ ] Health checks pass
- [ ] No regressions in existing functionality

### Commit Strategy
- Atomic commits per component (e.g., "feat(messagebus): add NATS container and docker-compose config")
- Follow conventional commit format
- Include Co-Authored-By: Claude Sonnet 4.5
- Test after each commit

---

## Key Decisions

### Phase 1 Decisions (2026-02-18):
- **Message Bus Choice**: NATS with JetStream
  - Reasoning: Simpler than RabbitMQ, better performance, excellent Go client
- **Topic Structure**: `loom.{category}.{project_id}`
  - Examples: `loom.tasks.loom`, `loom.results.loom`, `loom.events.dispatch`
- **Message Format**: JSON for human readability (can optimize to protobuf later)
- **Durable Consumers**: Yes, to prevent message loss on restart
- **Connection Strategy**: Single NATS connection per service, multiplexed

---

## Success Criteria

### Phase 1 Success Metrics:
- [ ] NATS container runs and is healthy
- [ ] Control plane can publish messages to NATS
- [ ] Control plane can subscribe to NATS topics
- [ ] Messages are durable (survive container restart)
- [ ] Health endpoint shows NATS connection status
- [ ] No increase in API latency
- [ ] Existing bead operations still work

### Overall Architecture Success:
- [ ] Task dispatch latency < 100ms
- [ ] Message throughput > 10,000 msgs/sec
- [ ] No message loss (persistent queues)
- [ ] Horizontal scaling of all services
- [ ] Service-level metrics (RED method)

---

## Notes

### Current Implementation Status:
- ✅ Git-centric bead storage implemented
- ✅ Working directory fix deployed
- ✅ System operational
- ⚠️ API showing slowness (may be fixed by architecture improvements)

### Architecture References:
- `docs/MICROSERVICES_ARCHITECTURE_REVIEW.md` - Full architecture analysis and plan
- Message Bus: NATS Documentation (https://docs.nats.io/)
- gRPC Best Practices: https://grpc.io/docs/guides/performance/
- Microservices Patterns: https://microservices.io/patterns/index.html

---

*This plan follows the EPCC workflow and is maintained by the development AI agent.*
