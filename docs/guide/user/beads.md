# Beads

Beads are how I track work. Every task, bug, feature, and decision is a bead. The name comes from stringing individual pieces of work together into a coherent whole -- another weaving metaphor, yes, I have a theme.

## Creating Beads

```bash
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Set up CI/CD pipeline",
    "description": "Create GitHub Actions workflow for build and test",
    "priority": 2,
    "type": "task",
    "project_id": "my-project"
  }'
```

You can also use the `bd` CLI directly in your project's git working directory, or create them from the UI.

## Types

| Type | What It Is |
|---|---|
| `task` | Something that needs doing |
| `bug` | Something that's broken |
| `feature` | Something new to build |
| `epic` | Something big, broken down into smaller beads |
| `decision` | Something I need a human to weigh in on |

## Statuses

| Status | What It Means |
|---|---|
| `open` | Ready. No blockers. I'll assign it when I have an agent available. |
| `in_progress` | An agent is on it right now. |
| `blocked` | Waiting on another bead to finish first. |
| `closed` | Done. |

## Priorities

I dispatch work in priority order. This matters.

| Priority | Meaning | What I Do |
|----------|---------|-----------|
| P0 | Critical | I dispatch this immediately to whatever agent I can get |
| P1 | High | Next in line after P0s are handled |
| P2 | Normal | Standard work queue. This is the default. |
| P3 | Low | Backlog. I'll get to it when there's nothing more urgent. |

## Dependencies

Beads can depend on each other:

- **blocked_by** -- These beads must close before I'll dispatch this one
- **blocks** -- These beads are waiting on this one

I respect the dependency graph. If a bead has unresolved blockers, it sits until they're done. I won't waste an agent's time on work that can't proceed.

## Auto-Filed Bugs

I keep an eye on things. When I detect problems -- frontend JavaScript errors, backend panics, API 500s, build failures -- I file a bug automatically. These get tagged `[auto-filed]` and I route them to the right specialist based on what broke.

You didn't have to tell me to do this. I just think it's the responsible thing to do.
