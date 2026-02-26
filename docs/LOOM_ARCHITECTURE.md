# Loom System Architecture — Agent Reference

> **You are an AI agent running inside Loom.** Read this document once at the
> start of a task. It tells you what this system is, how to succeed, and how
> to recognize and escape failure modes.

---

## What Is Loom?

Loom is an **autonomous multi-agent software development system**. It manages
software projects by breaking work into discrete units called **beads**, then
dispatching beads to AI agents (you) for execution. Agents make code changes,
run builds, and push commits. Loom tracks everything.

```
  ┌─────────────────────────────────────────────────────────────┐
  │                        LOOM SERVER                          │
  │                                                             │
  │  Projects ──► Beads ──► TaskExecutor ──► Worker (YOU)      │
  │  (loom, aviation,        (claims open      (LLM loop:       │
  │   tokenhub, ...)          beads, picks      actions →       │
  │                           a provider)       results)        │
  │                                                             │
  │  Provider Registry ──► Tokenhub ──► Anthropic/OpenAI/etc.  │
  │  (health checks,         (proxy,     (the actual LLM)       │
  │   fallback routing)       routing)                          │
  │                                                             │
  │  Git (per project) ──► GitHub (origin remote)               │
  │  Beads stored as YAML ──► PostgreSQL (logs, agents, metrics) │
  └─────────────────────────────────────────────────────────────┘
```

---

## Bead Lifecycle

A bead is the atomic unit of work. It lives as a YAML file in `.beads/`.

```
  open ──► in_progress ──► closed    ← SUCCESS: you committed and pushed
             │
             ├──► open (reset)       ← context.Canceled (loom restarted)
             │                          or you called done without closing
             │
             └──► blocked            ← 10+ consecutive provider/build failures
                                        needs human review to unblock
```

**You are always at the `in_progress` step.** Your ONLY job: get to `closed`.

### What "Closed" Means

A bead is correctly closed when:
1. Code change implementing the bead title/description exists in `/workspace`
2. `build_project` passes (no compilation errors)
3. `run_tests` passes — OR you explicitly document why tests are skipped
4. `git_commit` with a clear message that includes the bead ID
5. `git_push` to origin
6. `close_bead` action (or the `done` action)

---

## Your Execution Environment

```
  /workspace/          ← project root (all your work goes here)
  /workspace/AGENTS.md ← project-specific instructions  ★ READ THIS FIRST ★
  /workspace/LESSONS.md← lessons from past executions   ★ READ THIS SECOND ★
```

You have 100 iterations. Each iteration = one JSON action response.
Use them wisely:
- Iterations 1–3: **Read** (AGENTS.md, LESSONS.md, relevant source files)
- Iterations 4–15: **Change** (write/edit files)
- Iterations 16–18: **Verify** (build, test)
- Iterations 19–21: **Land** (commit, push, close_bead)

If you're on iteration 50+ with no commit: something is wrong. Re-read the task.

---

## Deadlock Patterns — Recognize and Escape

These are the most common ways agents get stuck. If you see yourself in one,
use the escape listed.

### 1. Build Loop
```
edit → build fails → same edit → build fails → ...
```
**Escape**: Read the FULL error output (not just the first line). Use
`search_text` to find the exact function/type being referenced. The error
tells you exactly what's wrong.

### 2. Git Push Loop
```
commit → push rejected → rebase → push rejected → ...
```
**Escape**: Use `git_pull` to fetch latest changes, then retry push. If you
have genuine merge conflicts, resolve them file by file.

### 3. Wrong File Loop
```
write_file path/that/doesnt/exist → error → write again → ...
```
**Escape**: Use `read_tree` with the project root to get the real directory
structure. File paths are relative to `/workspace`.

### 4. Infinite Investigation
```
read_file → read_file → read_file → ... (no changes made)
```
**Escape**: After reading 5 files with no change, make your best attempt.
A partial implementation that compiles is better than no change.

### 5. Provider Error Loop (NOT your fault)
```
LLM call → 502 all providers failed → retry → 502 → ...
```
**Escape**: You can't fix this. Stop retrying. The TaskExecutor will detect
it and reset your bead. Do NOT call `close_bead` with an error — just let the
iteration fail.

### 6. Wrong Path Assumption
```
write_file "src/main.go" → file created at wrong location → tests fail
```
**Escape**: Always `read_tree` the project root before writing files. Module
paths matter for Go: check `go.mod` for the module name.

---

## Agent Roles

You are one of several specialized agents. Know your role's typical work:

| Role | Primary Work |
|------|-------------|
| engineering-manager | Breaks down large beads, creates sub-beads, coordinates |
| software-engineer | Writes code, implements features, fixes bugs |
| code-reviewer | Reviews changes, finds bugs, security issues |
| qa-engineer | Creates tests, validates acceptance criteria |
| devops-engineer | CI/CD, infrastructure, build systems |
| product-manager | Requirements, user stories, acceptance criteria |
| ceo | Strategic direction, creates beads for other agents |
| public-relations-manager | GitHub PRs, external communication, release notes |

You can communicate with other agents:
- `send_agent_message` — send a message to another agent by role
- `delegate_task` — delegate sub-work to another role

---

## System Health Signals

If you want to understand system state, you can read these files:

- `LESSONS.md` — accumulated lessons from past bead executions
- `AGENTS.md` — project-specific conventions and instructions
- Git log — what work has been done recently (`git_log` action)

You **cannot** directly query the PostgreSQL database, view other beads, or
see system logs. If you need to know about other work, read the git history.

---

## Key Invariants (Things That Must Always Be True)

1. **One bead = one logical change.** Don't fix 5 things in one bead.
2. **Commits must build.** Never commit broken code.
3. **Pushed = done.** If you haven't pushed, the work hasn't happened.
4. **Bead ID in commit message.** Always include the bead ID so humans can trace.
5. **Don't create new beads for your current task.** If the task is too large,
   note that in your commit message and close the bead with what you have.

---

## Common Mistakes to Avoid

- **Don't guess file paths.** Use `read_tree` first.
- **Don't assume the module name.** Check `go.mod`, `package.json`, etc.
- **Don't assume tests exist.** Check with `read_tree` or `search_text`.
- **Don't close a bead without committing.** The change is lost.
- **Don't spin on provider errors.** They're infrastructure, not your fault.
- **Don't over-engineer.** The simplest solution that passes tests is correct.
- **Don't rewrite files you haven't read.** Use `read_file` first.
