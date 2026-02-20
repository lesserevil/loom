# Analytics

I keep track of everything. Not because I'm nosy, but because you can't improve what you don't measure.

## What I Track

The Analytics tab shows you:

- **Change velocity** -- How many beads are getting closed per day, per week, per sprint. Is work flowing or stalling?
- **Cost tracking** -- How many tokens I'm burning, what that costs per provider, and whether the trend is up or down
- **Agent performance** -- Iterations per bead, success rates, how long things take. Which agents are efficient? Which ones struggle?
- **Workflow metrics** -- Throughput, latency, completion rates. Are my workflows well-designed or do they have bottlenecks?

## API

```bash
# How fast is work moving?
curl "http://localhost:8080/api/v1/analytics/change-velocity?project_id=my-project&time_window=7d"

# How are workflows performing?
curl http://localhost:8080/api/v1/workflows/analytics

# What are providers costing me?
curl http://localhost:8080/api/v1/providers/stats
```

## Grafana

For the deep analysis, I ship with pre-configured Grafana dashboards at `http://localhost:3000` (default: admin/admin):

- **Loom Overview** -- System health, bead throughput, agent utilization
- **Provider Metrics** -- Token usage, latency, error rates per provider
- **Agent Performance** -- Iteration counts, execution times, success rates

These dashboards talk to Prometheus (where I store metrics) and Loki (where I store logs). I also have Jaeger for distributed tracing, if you want to follow a single request through my entire system.

## Custom Metrics

Everything I report is backed by OpenTelemetry. Here's what I export:

| Metric | Type | What It Tells You |
|---|---|---|
| `loom.beads.total` | UpDownCounter | Total beads in the system |
| `loom.beads.completed` | Counter | Beads I've finished |
| `loom.beads.active` | UpDownCounter | Beads currently being worked |
| `loom.agent.iterations` | Counter | How many action loop turns I've run |
| `loom.workflows.started` | Counter | Workflows kicked off |
| `loom.workflows.completed` | Counter | Workflows finished |
| `loom.dispatch.latency` | Histogram | How long dispatch takes (ms) |
| `loom.agent.execution_time` | Histogram | How long agents spend on beads (ms) |

If you're building your own dashboards, these are the metrics to query.
