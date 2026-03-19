# Quick Start Guide

Get Loom running and your first agents working in under 10 minutes.

## Prerequisites

- Docker and Docker Compose
- An OpenAI-compatible LLM endpoint (see below)

## 1. Choose a Provider

Loom talks to any OpenAI-compatible LLM endpoint. Point `LOOM_PROVIDER_URL` at whichever you have:

| Provider | Example URL |
|----------|-------------|
| [OpenAI](https://platform.openai.com) | `https://api.openai.com/v1` |
| [Ollama](https://ollama.com) (local) | `http://host.docker.internal:11434/v1` |
| [vLLM](https://docs.vllm.ai) (local GPU) | `http://host.docker.internal:8000/v1` |
| [LiteLLM](https://docs.litellm.ai) proxy | `http://host.docker.internal:4000/v1` |
| [TokenHub](https://github.com/jordanhubbard/tokenhub) | `http://host.docker.internal:8090/v1` |
| Any OpenAI-compatible API | `https://your-provider.example.com/v1` |

## 2. Start Loom

```bash
git clone https://github.com/jordanhubbard/loom.git
cd loom
cp .env.example .env
# Edit .env and set LOOM_PROVIDER_URL (and LOOM_PROVIDER_API_KEY if needed)
make start
```

This builds the containers and starts the full stack (Loom, PostgreSQL, NATS).
Wait about 30 seconds for everything to initialize, then open:

- **Loom UI**: http://localhost:8080

## 3. Set Up a Provider

Loom needs at least one LLM provider registered to power its agents. You can do this
via the UI (**Providers** tab) or the API:

```bash
curl -X POST http://localhost:8080/api/v1/providers \
    -H 'Content-Type: application/json' \
    -d '{
        "id": "my-provider",
        "name": "My LLM",
        "type": "openai",
        "endpoint": "https://api.openai.com/v1",
        "model": "gpt-4o",
        "api_key": "your-api-key-here"
    }'
```

### Verify the Provider

Within 30 seconds, the heartbeat will check your provider. Verify it's healthy:

```bash
curl -s http://localhost:8080/api/v1/providers | jq '.[].status'
```

You should see `"healthy"`. If you see `"failed"`, check the error:

```bash
curl -s http://localhost:8080/api/v1/providers | jq '.[].last_heartbeat_error'
```

## 4. Add a Project

Navigate to **Projects** in the Loom UI and click **Add Project**, or use the API:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
    -H 'Content-Type: application/json' \
    -d '{
        "name": "My App",
        "git_repo": "git@github.com:youruser/yourrepo.git",
        "branch": "main",
        "beads_path": ".beads"
    }'
```

### Add the Deploy Key

Loom generates a unique SSH keypair for each project. Retrieve the public key:

```bash
curl -s http://localhost:8080/api/v1/projects/<project-id>/git-key | jq -r '.public_key'
```

Add this key as a **deploy key with write access** in your git hosting service:

- **GitHub**: Repository Settings > Deploy keys > Add deploy key
- **GitLab**: Settings > Repository > Deploy keys

Loom will clone the repository on the next dispatch cycle.

## 5. Use the CEO Dashboard

Open http://localhost:8080 and click **CEO Dashboard**. This is your command center.

The CEO Dashboard shows:
- Provider health and agent status
- Open beads across all projects
- The **Ask Loom** prompt for directing agents

### File Your First Beads

Use the **Ask Loom** prompt to give instructions, or create beads directly:

```bash
# Install the beads CLI (optional but recommended)
# See: https://github.com/steveyegge/beads

# Or create beads via the API:
curl -X POST http://localhost:8080/api/v1/beads \
    -H 'Content-Type: application/json' \
    -d '{
        "title": "Set up CI/CD pipeline",
        "description": "Create GitHub Actions workflow for build and test",
        "priority": 2,
        "type": "task",
        "project_id": "<project-id>"
    }'
```

### Priority Levels

| Priority | Meaning | Agent Behavior |
|----------|---------|----------------|
| P0 | Critical | Dispatched immediately to any available agent |
| P1 | High | Dispatched next after P0 work |
| P2 | Normal | Standard work queue (default) |
| P3 | Low | Backlog, picked up when nothing higher exists |

## 6. Watch Agents Work

Once you have a healthy provider and open beads, Loom's agents automatically:

1. **Claim** beads matching their expertise
2. **Read** your codebase to understand context
3. **Execute** actions (read files, search code, write changes, run tests)
4. **Iterate** through a multi-turn action loop (up to 15 turns per dispatch)
5. **Complete** or **escalate** when done

Monitor progress:

```bash
# See which agents are working
curl -s http://localhost:8080/api/v1/agents | jq '.[] | {name, status, current_bead}'

# Follow container logs
make logs
```

The agents work autonomously. Each agent has a persona (Engineering Manager,
Code Reviewer, DevOps Engineer, etc.) that determines what beads it picks up
and how it approaches the work.

## What's Next

- [User Guide](../guide/user/index.md) -- Learn the web UI
- [Administrator Guide](../guide/admin/index.md) -- Configuration, deployment, and operations
- [Developer Guide](../guide/developer/index.md) -- Architecture and contributing

## Quick Reference

```bash
make start      # Start Loom (Docker)
make stop       # Stop Loom
make restart    # Rebuild and restart
make logs       # Follow container logs
make test       # Run tests locally
make help       # All available commands
```
