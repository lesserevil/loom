# Conversations

Most of the time, my agents work independently. You file a bead, I dispatch it, the agent handles it. But sometimes you want to think through a problem together with one of my agents. That's what pair mode is for.

## Pair Programming

Pair-programming mode is a real-time conversation between you and an agent, scoped to a specific bead. The agent has full context of the bead and can execute actions (reading files, writing code, searching the codebase) while you talk.

To start:

1. Open a bead in the Bead Viewer
2. Click **Pair**
3. Pick an agent from the dropdown
4. Start talking

The agent responds via streaming -- you'll see the response build in real-time.

## When to Use This

- **Before work starts** -- When requirements are fuzzy and you want to hash them out before the agent goes off on its own
- **During implementation** -- When you have opinions about the approach and want to collaborate rather than delegate
- **During review** -- When you're looking at what an agent did and want to discuss changes
- **When debugging** -- Two heads (even if one is artificial) are better than one

## Persistence

I save the chat history for each bead. Come back later and the conversation is still there. You don't lose context between sessions.

## API

```bash
# Start a pair session (returns a Server-Sent Events stream)
curl -N -X POST http://localhost:8080/api/v1/pair \
  -H "Content-Type: application/json" \
  -d '{
    "bead_id": "bead-123",
    "agent_id": "agent-456",
    "message": "How should we approach the auth middleware?"
  }'

# See all conversations for a project
curl http://localhost:8080/api/v1/conversations?project_id=my-project
```
