# Git & Deploy Keys

Loom interacts with git repositories through per-project SSH deploy keys.

## How It Works

1. When you create a project, Loom generates a unique Ed25519 SSH keypair
2. You add the public key as a deploy key (with write access) in your git host
3. Loom uses the private key for all git operations (clone, pull, push)

## Setting Up Deploy Keys

Retrieve the public key:

```bash
curl -s http://localhost:8080/api/v1/projects/<id>/git-key | jq -r '.public_key'
```

Add it as a deploy key with **write access**:

- **GitHub**: Repository > Settings > Deploy keys > Add deploy key
- **GitLab**: Settings > Repository > Deploy keys
- **Bitbucket**: Repository settings > Access keys

## Git Operations

Loom performs these git operations automatically:

| Operation | When |
|---|---|
| `clone` | First dispatch for a project |
| `pull` | Before each dispatch cycle |
| `commit` | After agent completes code changes |
| `push` | After successful commit |

## Security Model

- Private keys are encrypted at rest using the master password
- Each project has its own keypair (compromise of one doesn't affect others)
- Deploy keys have repository-scoped access only
- All SSH operations use strict host key checking

## Manual Git Operations

From the Projects tab, use the Git Operations button to manually:

- **Pull** -- Update from remote
- **Push** -- Push local changes
- **Status** -- Check working directory status
- **Diff** -- View uncommitted changes
