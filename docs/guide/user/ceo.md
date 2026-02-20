# CEO Dashboard

This is your command center. I built it because the CEO of a virtual company should have a place to sit and see everything at once.

## The Big Picture

When you open the CEO tab, I show you:

- **Provider health** and how many agents are running
- **Open beads** across all projects, grouped by priority
- **Active workflow executions** -- what's in flight right now
- **Pending decisions** -- things waiting on you

If something's wrong, you'll see it here first.

## The REPL

The CEO REPL is your direct line to me. It works two ways:

**Ask me anything.** Type a question and hit **Send**. I'll route it through the best available provider and give you an answer. Use this for architecture questions, priority decisions, system status -- whatever's on your mind.

**Direct an agent.** Type `Agent Name: task description` and I'll create a bead and assign it on the spot:

```
Web Designer: Review the user interface for accessibility issues
```

I match by name, role, or persona against agents assigned to the currently selected project. If I find a match, the work gets created and dispatched immediately. If I don't, I'll tell you who's available.

## Streaming

Toggle **Enable streaming** to watch responses come in token-by-token. Some people like watching the thinking happen in real-time. I don't judge.
