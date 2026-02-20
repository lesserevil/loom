# Your First Project

This guide walks you through adding your first project to Loom, from git setup through your first autonomous agent run.

## Prerequisites

- Loom is running (see [Quick Start](quickstart.md))
- At least one healthy LLM provider registered
- A git repository you want Loom to work on

## 1. Add the Project

Navigate to the **Projects** tab in the Loom UI and click **Add Project**.

Fill in:

- **Name**: A human-readable name (e.g., "My Web App")
- **Git URL**: The SSH clone URL (e.g., `git@github.com:youruser/myapp.git`)
- **Branch**: The branch to work on (default: `main`)

Or use the API:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
    -H 'Content-Type: application/json' \
    -d '{
        "name": "My Web App",
        "git_repo": "git@github.com:youruser/myapp.git",
        "branch": "main"
    }'
```

## 2. Configure the Deploy Key

Loom generates a unique SSH keypair per project. Retrieve the public key:

```bash
curl -s http://localhost:8080/api/v1/projects/<project-id>/git-key | jq -r '.public_key'
```

Add this key as a **deploy key with write access** in your git hosting service:

- **GitHub**: Repository > Settings > Deploy keys > Add deploy key
- **GitLab**: Settings > Repository > Deploy keys

## 3. Verify Git Connectivity

Once the deploy key is added, Loom will clone the repository on its next dispatch cycle (every 10 seconds). You can check the status:

```bash
curl -s http://localhost:8080/api/v1/projects/<project-id> | jq '.git_status'
```

## 4. Create Agents

Loom auto-creates agents from personas when work needs to be done, but you can also create them explicitly:

```bash
curl -X POST http://localhost:8080/api/v1/agents \
    -H 'Content-Type: application/json' \
    -d '{
        "name": "My Engineer",
        "persona": "default/engineering-manager",
        "project_id": "<project-id>"
    }'
```

## 5. File Your First Bead

Beads are Loom's work items. Create one to give your agents something to do:

```bash
curl -X POST http://localhost:8080/api/v1/beads \
    -H 'Content-Type: application/json' \
    -d '{
        "title": "Add README with project description",
        "description": "Create a comprehensive README.md for the project",
        "priority": 2,
        "type": "task",
        "project_id": "<project-id>"
    }'
```

## 6. Watch It Work

The dispatcher will automatically assign the bead to a matching agent. Monitor progress:

- **Web UI**: Watch the bead move through kanban columns (Open -> In Progress -> Done)
- **Logs**: `make logs` to see agent activity
- **API**: `curl http://localhost:8080/api/v1/beads?project_id=<id>` to check status

## Next Steps

- [User Guide](../guide/user/index.md) -- Learn all the UI features
- [Beads](../guide/user/beads.md) -- Understand bead types, priorities, and workflows
- [Agents & Personas](../guide/user/agents.md) -- Customize agent behavior
