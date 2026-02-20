# Scaling

Loom scales horizontally through agent replicas and vertically through provider capacity.

## Agent Scaling

### Docker Compose

```bash
make scale-coders N=3
make scale-reviewers N=2
make scale-qa N=2
# Or all at once:
make scale-agents CODERS=3 REVIEWERS=2 QA=2
```

### Kubernetes

```bash
kubectl scale deployment loom-agent-coder -n loom --replicas=3
```

## Provider Scaling

Add multiple providers to increase LLM throughput. Loom load-balances across healthy providers using weighted round-robin.

## Database Scaling

- **PgBouncer** connection pooler is included by default (max 100 client connections, 20 pool size)
- PostgreSQL can be scaled to a managed service (RDS, Cloud SQL) by updating connection settings

## Message Bus

NATS with JetStream handles inter-service messaging. For high-throughput:

- Enable NATS clustering for HA
- Increase JetStream storage limits
- Configure consumer concurrency per agent type

## Observability at Scale

All services export metrics to Prometheus via OpenTelemetry:

- Monitor agent utilization to right-size replicas
- Track provider latency to identify bottlenecks
- Use Grafana dashboards for real-time visibility
