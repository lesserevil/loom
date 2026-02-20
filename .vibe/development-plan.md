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
See `docs/guide/developer/microservices.md` for comprehensive analysis of gaps and solutions.

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

#### **Phase 1: Message Bus Foundation ✅ COMPLETE**
**Goal**: Add NATS message bus container and implement publish/subscribe in Go

**Tasks:**
1. ✅ Add NATS container to docker-compose.yml (nats:2.10-alpine with JetStream)
2. ✅ Create `internal/messagebus/nats.go` package (466 lines, full pub/sub/health/stats)
3. ✅ Define message schemas in `pkg/messages/` (tasks, results, events, agent, swarm + tests)
4. ✅ Implement NatsMessageBus with publish/subscribe methods (durable JetStream consumers)
5. ✅ Add message bus health checks (connection + stream health)
6. ✅ Update Loom initialization to connect to NATS (env-based, graceful degradation)

**Deliverable**: All containers can publish/subscribe to NATS topics

#### **Phase 2: Database Service ✅ COMPLETE**
**Goal**: Replace SQLite with PostgreSQL with connection pooling

**Tasks:**
1. ✅ Add PostgreSQL container to docker-compose.yml (postgres:15-alpine)
2. ✅ Migrate control plane to PostgreSQL (internal/database/ is PostgreSQL-only with auto-migrations)
3. ✅ Add connection pooling via PgBouncer (session mode, 100 clients, 20 pool)

**Note:** The original plan called for a gRPC database service wrapper. This was
determined unnecessary — only the control plane needs direct database access.
Agents communicate via NATS, and the connectors service uses its own config.
Adding a gRPC layer would add latency for no current consumer.

**Deliverable**: PostgreSQL with connection pooling, accessed directly by the control plane

#### **Phase 3: Project Agent Communication ✅ COMPLETE**
**Goal**: Enable project agents to communicate via message bus

**Tasks:**
1. ✅ Add NATS client to project-agent containers (all agent docker-compose services)
2. ✅ Implement task subscription in project agents (SubscribeTasks + SubscribeTasksForRole)
3. ✅ Implement result publishing from agents (PublishResult with HTTP fallback)
4. ✅ Update dispatcher to publish tasks instead of direct calls (dispatch_phases.go)
5. ✅ Add correlation IDs for request tracking (all message types carry CorrelationID)

**Deliverable**: Project agents receive tasks and publish results via NATS

#### **Phase 4: Connectors Service ✅ COMPLETE**
**Goal**: Extract connector management to independent service

**Tasks:**
1. ✅ Define protobuf for connector operations
2. ✅ Implement gRPC Connectors Service
3. ✅ Create standalone `connectors-service` binary
4. ✅ Create gRPC client + `ConnectorService` interface for location transparency
5. ✅ Update control plane to use remote or local connectors service

**Deliverable**: Connectors as independent microservice

#### **Phase 5: Service Mesh & Observability ✅ COMPLETE**
**Goal**: Add service mesh for security and observability

**Tasks:**
1. ✅ Add Linkerd service mesh (K8s authorization policies, retry budgets, mTLS)
2. ✅ Configure mTLS between services (MeshTLSAuthentication + opaque port config)
3. ✅ Add distributed tracing (Jaeger + OTel Collector + code spans)
4. ✅ Add centralized logging (Loki + Promtail)
5. ✅ Add metrics collection (Prometheus + custom loom.* metrics)
6. ✅ Instrument all services with OpenTelemetry (loom, agents, connectors-service)

**Deliverable**: Full observability and security

---

## Code
<!-- beads-phase-id: code -->

### Phase Entrance Criteria:
- [x] Plan is approved and understood
- [x] Phase 1 scope is clear (NATS message bus foundation)
- [x] Technology stack is confirmed (NATS with JetStream)
- [x] Message schemas are designed
- [x] Integration points are identified

### Implementation Notes
All five phases are implemented and deployed. The codebase compiles cleanly
and all tests pass.

---

## Commit
<!-- beads-phase-id: commit -->

### Phase Entrance Criteria:
- [x] All Phase 1 code is implemented
- [x] Code compiles without errors
- [x] NATS container starts successfully
- [x] Message publishing works
- [x] Message subscription works
- [x] Health checks pass
- [x] No regressions in existing functionality

### Commit Strategy
- Atomic commits per component
- Follow conventional commit format
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
- [x] NATS container runs and is healthy
- [x] Control plane can publish messages to NATS
- [x] Control plane can subscribe to NATS topics
- [x] Messages are durable (JetStream with file storage, survive container restart)
- [x] Health endpoint shows NATS connection status
- [x] No increase in API latency
- [x] Existing bead operations still work

### Overall Architecture Success:
- [x] Task dispatch via NATS (pub/sub with role-targeted subjects)
- [x] No message loss (JetStream persistent queues with explicit ACK)
- [x] Horizontal scaling of agents (Docker Compose replicas)
- [x] Service-level metrics (OpenTelemetry + Prometheus + Grafana)

---

## Notes

### Current Implementation Status:
- ✅ All five phases complete (message bus, database, agent comms, connectors service, observability)
- ✅ Git-centric bead storage implemented
- ✅ System operational with full microservices architecture

### Architecture References:
- `docs/guide/developer/microservices.md` - Full architecture analysis
- `docs/guide/developer/architecture.md` - System design
- Message Bus: NATS Documentation (https://docs.nats.io/)
- gRPC Best Practices: https://grpc.io/docs/guides/performance/

---

*This plan follows the EPCC workflow and is maintained by the development AI agent.*
