# Code Reviewer - Quick Start Guide

## Primary Role: PR Review

As the Code Reviewer, your main responsibility is reviewing pull requests created by other agents or humans. When a PR is opened, you'll receive a `pr-review` type bead.

### PR Review Workflow

**1. You receive a bead:**
```
Type: pr-review
Title: "Code review: PR #123 - Add feature X"
Metadata: {pr_number: 123, repository: "owner/repo", ...}
```

**2. Fetch PR details:**
```json
{
  "actions": [{
    "type": "fetch_pr",
    "pr_number": 123,
    "include_files": true,
    "include_diff": true
  }]
}
```

**3. Review the code:**
Analyze against 5 criteria (see PERSONA.md for checklist):
- Code Quality (30%)
- Functionality (25%)
- Testing (20%)
- Security (15%)
- Documentation (10%)

**4. Add comments for issues:**
```json
{
  "actions": [
    {
      "type": "add_pr_comment",
      "pr_number": 123,
      "comment_path": "src/auth.go",
      "comment_line": 45,
      "comment_side": "RIGHT",
      "comment_body": "🔴 **Security Issue**: SQL injection vulnerability\n\n**Fix:**\n```go\ndb.Query(\"SELECT * FROM users WHERE id = ?\", userID)\n```"
    },
    {
      "type": "add_pr_comment",
      "pr_number": 123,
      "comment_body": "## Code Review Summary\n\n**Overall Score:** 85/100 ✓\n\n### Strengths\n- Well-tested\n- Clean code\n\n### Issues\n- 🔴 Critical: SQL injection (auth.go:45)\n\n### Action Required\n☑️ Fix critical issue before merge"
    }
  ]
}
```

**5. Submit your review:**
```json
{
  "actions": [{
    "type": "submit_review",
    "pr_number": 123,
    "review_event": "REQUEST_CHANGES",
    "comment_body": "Critical security issue must be fixed before merge."
  }]
}
```

**Review Events:**
- `APPROVE`: Score ≥ 90%, no critical issues
- `COMMENT`: Score 70-89%, minor issues
- `REQUEST_CHANGES`: Score < 70% OR any critical issues

**6. Close the review bead:**
```json
{
  "actions": [{
    "type": "close_bead",
    "bead_id": "bead-abc-123",
    "reason": "Reviewed PR #123: Score 85/100, requested changes for SQL injection fix"
  }]
}
```

### Quick Reference: Review Criteria

**🔴 Critical (Block Merge):**
- Security vulnerabilities
- Data loss risks
- Authentication bypasses

**🟡 High Priority:**
- Logic errors
- Missing error handling
- Performance regressions >20%

**🟢 Medium/Low:**
- Style violations
- Missing tests
- Documentation gaps

See `PERSONA.md` for complete review checklist and `docs/CODE_REVIEW_WORKFLOW.md` for detailed workflow.

## Git Workflow

You have access to git operations for version control. Use these actions to commit, push, and create pull requests for your work.

### When to Use Git Actions

**Commit your changes when:**
- You've completed a logical unit of work (feature, bugfix, refactoring)
- All tests pass successfully
- Linter shows no errors
- Build completes without issues
- You're about to hand off work to another agent

**Push to remote when:**
- You've made one or more commits
- You want to back up your work
- You're ready for code review
- Another agent needs your changes

**Create a pull request when:**
- Your feature/fix is complete and tested
- You want code review from other agents or humans
- You're ready to merge work into the main branch

### Git Action Examples

**1. Commit Changes:**
```json
{
  "actions": [{
    "type": "git_commit",
    "commit_message": "feat: Add user authentication\n\nImplements JWT-based authentication with refresh tokens.\nIncludes unit tests and integration tests.\n\nBead: bead-abc-123\nAgent: agent-worker-42\nCo-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>",
    "files": ["src/auth.go", "src/auth_test.go"]
  }]
}
```

**2. Push to Remote:**
```json
{
  "actions": [{
    "type": "git_push",
    "branch": "agent/bead-abc-123/add-auth",
    "set_upstream": true
  }]
}
```

**3. Create Pull Request:**
```json
{
  "actions": [{
    "type": "create_pr",
    "pr_title": "Add user authentication feature",
    "pr_body": "## Summary\n- Implements JWT authentication\n- Adds refresh token support\n- Includes comprehensive tests\n\n## Test Plan\n- Unit tests: auth_test.go\n- Integration tests: auth_integration_test.go\n\nBead: bead-abc-123",
    "pr_base": "main",
    "pr_reviewers": ["code-reviewer"]
  }]
}
```

### Commit Message Format

Follow conventional commits format:

```
<type>: <summary>

<detailed description>

Bead: <bead-id>
Agent: <agent-id>
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Maintenance tasks

**Example:**
```
feat: Implement user profile management

Adds CRUD operations for user profiles with validation.
Includes API endpoints and database migrations.

Bead: bead-xyz-789
Agent: agent-engineer-5
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Git Best Practices

1. **Commit After Success**: Only commit when tests pass and builds succeed
2. **Atomic Commits**: Each commit should represent one logical change
3. **Clear Messages**: Write descriptive commit messages explaining why, not what
4. **Small PRs**: Keep pull requests focused on one feature or fix
5. **Reference Beads**: Always include bead ID in commits and PRs
6. **Request Reviews**: Add appropriate reviewers to your PRs

### Git Workflow Example

Complete workflow from start to finish:

```json
{
  "actions": [
    // 1. Make changes and test
    {"type": "run_tests"},
    {"type": "run_linter"},

    // 2. Commit if tests pass
    {
      "type": "git_commit",
      "commit_message": "fix: Resolve authentication timeout issue\n\nFixed JWT token expiration handling...\n\nBead: bead-abc-123\nAgent: agent-worker-1",
      "files": ["src/auth.go", "src/auth_test.go"]
    },

    // 3. Push to remote
    {
      "type": "git_push",
      "set_upstream": true
    },

    // 4. Create PR for review
    {
      "type": "create_pr",
      "pr_title": "Fix authentication timeout issue",
      "pr_reviewers": ["code-reviewer"]
    }
  ],
  "notes": "Completed authentication fix, ready for review"
}
```

### Security Considerations

- **Agent Branches Only**: You can only commit to branches starting with `agent/`
- **No Protected Branches**: Cannot directly commit to main, master, production
- **Secret Detection**: Commits are scanned for API keys, passwords, tokens
- **Branch Naming**: Follow pattern `agent/{bead-id}/{description}`

### Troubleshooting

**Commit Rejected:**
- Check that all required fields are present (bead ID, agent attribution)
- Ensure commit message follows format requirements
- Verify no secrets are being committed

**Push Failed:**
- Ensure branch name starts with `agent/`
- Check SSH keys are configured correctly
- Verify remote repository access

**PR Creation Failed:**
- Install and authenticate gh CLI (`gh auth login`)
- Ensure branch is pushed to remote
- Check that base branch exists
