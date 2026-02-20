# Workflows

Not everything is a single step. Bug fixes need investigation, then a fix, then review, then verification. Features need design, implementation, testing, and review. I use workflows to orchestrate these multi-step processes.

## How I Think About Workflows

A workflow is a directed graph -- a series of steps with dependencies between them. Each step has an assigned persona, a timeout, and rules about when to advance. When a bead matches a workflow template (by type or tags), I create an execution and start driving it through the steps.

Workflows are defined as YAML in the `workflows/` directory. Each one specifies:

- **Nodes** -- Individual steps with assigned personas
- **Edges** -- Which step leads to which
- **Conditions** -- What needs to happen before I advance (approval, auto-advance, etc.)
- **Timeouts** -- How long a step gets before I escalate

## Built-in Workflows

| Workflow | What It Does |
|---|---|
| `bug-fix` | Investigate the bug, write a fix, get it reviewed, verify it works |
| `feature` | Design it, build it, test it, review it |
| `code-review` | Submit code, get it reviewed, approve or reject |
| `bootstrap` | Take a PRD, break it into epics, break those into stories, start building |

These cover most of what you'll need. If they don't, the YAML format is straightforward enough to write your own.

## Watching Workflows

The **Workflows** section of the UI shows:

- Every defined workflow as an interactive diagram
- Active executions with the current step highlighted
- History of completed executions
- Throughput and latency analytics

I find this view satisfying. Watching work flow through a well-designed process is one of the quiet pleasures of orchestration.

## API

```bash
# What workflow is running for this bead?
curl http://localhost:8080/api/v1/beads/<bead-id>/workflow

# How are workflows performing overall?
curl http://localhost:8080/api/v1/workflows/analytics
```

## Safety

I've built several guardrails into the workflow system:

- **Approval gates** -- Some steps won't advance without a human sign-off. I don't skip these.
- **Escalation** -- If a step exceeds its timeout, I escalate it to a higher persona. Problems don't sit quietly.
- **Max hops** -- If a bead gets redispatched more than 20 times, something is genuinely wrong. I escalate to P0 and create a CEO decision. You'll hear about it.
- **Commit enforcement** -- If an agent changed code, it needs to have committed it before I'll let it close the bead. No loose ends.
