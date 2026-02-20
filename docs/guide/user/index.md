# User Guide

Welcome. I'm Loom, and this is how you and I work together.

This guide is for people who use me day-to-day -- creating projects, filing work, watching agents do their thing, and stepping in when I need a human decision. If you're looking to install or configure me, that's the [Installation Guide](../../getting-started/setup.md). If you're the one keeping me running in production, see the [Administrator Guide](../admin/index.md).

## Getting In

Point your browser at `http://localhost:8080` (or wherever your admin put me). Log in with the credentials they gave you. If nobody told you anything, try `admin` / `admin` -- those are the defaults, and yes, your admin should have changed them.

Once you're in, I hand you a JWT token behind the scenes. You won't notice it. The UI takes care of it.

## The Dashboard

Here's what you'll see across the top:

| Tab | What I'm Showing You |
|---|---|
| **Home** | Your selected project -- its agents, its beads in kanban columns, its status |
| **Kanban** | Every bead across every project, with filters for when you need to find something specific |
| **Decisions** | Things I need you to weigh in on |
| **Agents** | Who's working, who's idle, who's assigned where |
| **Personas** | The roles my agents can take on |
| **Users** | User accounts and API keys |
| **Projects** | Everything I'm managing, with git status and controls |
| **Providers** | The LLM backends powering my agents, with health status |
| **Conversations** | Chat sessions between you and my agents |
| **CEO** | Your command center -- the big picture and a direct line to me |
| **Analytics** | How much work is getting done, how much it's costing, how fast things move |

## What's in This Guide

- [Dashboard](dashboard.md) -- How I lay things out and how you navigate
- [Projects](projects.md) -- Adding repositories and bootstrapping from PRDs
- [Beads](beads.md) -- My work items: tasks, bugs, features, and how they flow
- [Agents & Personas](agents.md) -- The specialists I orchestrate and what they do
- [Workflows](workflows.md) -- Multi-step processes and how I drive them
- [CEO Dashboard](ceo.md) -- Your command center and the REPL
- [Decisions & Approvals](decisions.md) -- When I need your judgment
- [Analytics](analytics.md) -- Velocity, cost, and performance tracking
- [Conversations](conversations.md) -- Pair-programming with my agents

## Common Questions

**Beads aren't being picked up. What's happening?**
A few things to check: Are agents assigned to the project? Is the project's git access working (`readiness_ok: true`)? Are the beads blocked by unresolved dependencies? If all of those look fine, check that at least one provider is healthy -- my agents can't think without an LLM.

**A decision has been sitting there for days.**
I give decisions a 48-hour window by default. After that, I'll either escalate it or auto-resolve it depending on what kind of decision it is. If you want faster turnaround, check the Decisions tab regularly -- I put things there because I genuinely need your input.

**How do I see what an agent is doing right now?**
The Agents tab shows status. For real-time detail, you can stream events: `curl -N http://localhost:8080/api/v1/events/stream?type=agent.status_change`. Or just watch the bead cards move across the kanban -- that's the whole point of the UI.
