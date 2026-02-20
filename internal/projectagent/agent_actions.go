package projectagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/internal/github"
)

// executeCreatePR creates a pull request via the gh CLI tool.
// Params: title (required), body, base (default "main"), branch (auto-detected)
func (a *Agent) executeCreatePR(ctx context.Context, params map[string]interface{}) (string, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title parameter required")
	}

	body, _ := params["body"].(string)
	base, _ := params["base"].(string)
	if base == "" {
		base = "main"
	}

	args := []string{"pr", "create", "--title", title, "--base", base}
	if body != "" {
		args = append(args, "--body", body)
	} else {
		args = append(args, "--body", "Auto-generated PR from agent fix")
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = a.config.WorkDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// executeCreateBead creates a new bead on the control plane.
// Params: title (required), description, type (default "task"), priority (default 1), tags
func (a *Agent) executeCreateBead(ctx context.Context, params map[string]interface{}) (string, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title parameter required")
	}

	description, _ := params["description"].(string)
	beadType, _ := params["type"].(string)
	if beadType == "" {
		beadType = "task"
	}
	priority := 1
	if p, ok := params["priority"].(float64); ok {
		priority = int(p)
	}

	var tags []string
	if t, ok := params["tags"].([]interface{}); ok {
		for _, v := range t {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	payload := map[string]interface{}{
		"title":       title,
		"description": description,
		"type":        beadType,
		"priority":    priority,
		"project_id":  a.config.ProjectID,
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal bead request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/beads", a.config.ControlPlaneURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create bead: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("control plane returned %d: %v", resp.StatusCode, result)
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// executeVerify runs the project's test suite and returns results.
// Params: command (optional, default auto-detect), timeout (seconds, default 120)
func (a *Agent) executeVerify(ctx context.Context, params map[string]interface{}) (string, error) {
	command, _ := params["command"].(string)
	timeout := 120
	if t, ok := params["timeout"].(float64); ok {
		timeout = int(t)
	}

	if command == "" {
		command = a.detectTestCommand()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)
	cmd.Dir = a.config.WorkDir
	output, err := cmd.CombinedOutput()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Test command: %s\n", command))
	result.WriteString(fmt.Sprintf("Exit code: %d\n", cmd.ProcessState.ExitCode()))
	result.WriteString("---\n")
	result.WriteString(string(output))

	if err != nil {
		return result.String(), fmt.Errorf("tests failed: %w", err)
	}
	return result.String(), nil
}

// detectTestCommand tries to find the appropriate test command for the project.
func (a *Agent) detectTestCommand() string {
	checks := []struct {
		file    string
		command string
	}{
		{"Makefile", "make test 2>&1"},
		{"go.mod", "go test ./... 2>&1"},
		{"package.json", "npm test 2>&1"},
		{"Cargo.toml", "cargo test 2>&1"},
		{"pyproject.toml", "python -m pytest 2>&1"},
		{"requirements.txt", "python -m pytest 2>&1"},
	}

	for _, c := range checks {
		cmd := exec.Command("test", "-f", c.file)
		cmd.Dir = a.config.WorkDir
		if cmd.Run() == nil {
			return c.command
		}
	}
	return "echo 'No test framework detected'"
}

// ExportedCreateBead is a public wrapper for testing create_bead from integration tests.
func (a *Agent) ExportedCreateBead(ctx context.Context, params map[string]interface{}) (string, error) {
	return a.executeCreateBead(ctx, params)
}

// ExportedCloseBead is a public wrapper for testing close_bead from integration tests.
func (a *Agent) ExportedCloseBead(ctx context.Context, params map[string]interface{}) (string, error) {
	return a.executeCloseBead(ctx, params)
}

// ExportedVerify is a public wrapper for testing verify from integration tests.
func (a *Agent) ExportedVerify(ctx context.Context, params map[string]interface{}) (string, error) {
	return a.executeVerify(ctx, params)
}

// executeCloseBead closes a bead via the control plane API.
// Params: bead_id (required), reason (required)
func (a *Agent) executeCloseBead(ctx context.Context, params map[string]interface{}) (string, error) {
	beadID, _ := params["bead_id"].(string)
	if beadID == "" {
		return "", fmt.Errorf("bead_id parameter required")
	}
	reason, _ := params["reason"].(string)
	if reason == "" {
		return "", fmt.Errorf("reason parameter required")
	}

	payload := map[string]interface{}{
		"status": "closed",
		"context": map[string]string{
			"close_reason": reason,
			"closed_by":    "agent",
			"closed_at":    time.Now().UTC().Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal close request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/beads/%s", a.config.ControlPlaneURL, beadID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to close bead: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("control plane returned %d closing bead %s", resp.StatusCode, beadID)
	}

	return fmt.Sprintf("Bead %s closed: %s", beadID, reason), nil
}

// learnFromBashSuccess inspects a successful bash command and stores any
// recognized build/test/lint commands as project memory on the control plane.
func (a *Agent) learnFromBashSuccess(ctx context.Context, command string) {
	category := ""
	key := ""
	switch {
	case strings.HasPrefix(command, "go build") || strings.HasPrefix(command, "make build"):
		category = "build_system"
		key = "build_command"
	case strings.HasPrefix(command, "go test") || strings.HasPrefix(command, "npm test") ||
		strings.HasPrefix(command, "pytest") || strings.HasPrefix(command, "make test"):
		category = "build_system"
		key = "test_command"
	case strings.HasPrefix(command, "golangci-lint") || strings.HasPrefix(command, "eslint") ||
		strings.HasPrefix(command, "make lint"):
		category = "build_system"
		key = "lint_command"
	default:
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"value":      command,
		"confidence": 0.8,
	})
	url := fmt.Sprintf("%s/api/v1/projects/%s/memory/%s/%s",
		a.config.ControlPlaneURL, a.config.ProjectID, category, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// reportGitHubRepoURL auto-detects the git remote origin URL and PUTs it to
// the control plane as the project's github_repo. Safe to call in a goroutine;
// errors are logged but not fatal.
func (a *Agent) reportGitHubRepoURL(ctx context.Context) {
	ghc := github.NewClient(a.config.WorkDir, "")
	remoteURL, err := ghc.DetectRepoURL(ctx)
	if err != nil || remoteURL == "" {
		return
	}
	// Convert SSH remote (git@github.com:owner/repo.git) to owner/repo slug.
	slug := remoteURL
	if strings.Contains(slug, "github.com") {
		slug = strings.TrimSuffix(slug, ".git")
		if idx := strings.LastIndex(slug, "github.com"); idx >= 0 {
			slug = slug[idx+len("github.com"):]
			slug = strings.TrimPrefix(slug, "/")
			slug = strings.TrimPrefix(slug, ":")
		}
	}
	if slug == "" {
		return
	}
	payload, _ := json.Marshal(map[string]string{"github_repo": slug})
	url := fmt.Sprintf("%s/api/v1/projects/%s/github", a.config.ControlPlaneURL, a.config.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// ghClient returns a GitHub client for the agent's workspace.
func (a *Agent) ghClient() *github.Client {
	token := "" // gh CLI uses stored OAuth credentials by default
	return github.NewClient(a.config.WorkDir, token)
}

// executeGitHubListIssues lists open GitHub issues.
// Params: state (optional: "open", "closed", "all")
func (a *Agent) executeGitHubListIssues(ctx context.Context, params map[string]interface{}) (string, error) {
	state, _ := params["state"].(string)
	issues, err := a.ghClient().ListIssues(ctx, state)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(issues, "", "  ")
	return string(out), nil
}

// executeGitHubCreateIssue creates a GitHub issue.
// Params: title (required), body, labels ([]string)
func (a *Agent) executeGitHubCreateIssue(ctx context.Context, params map[string]interface{}) (string, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title parameter required")
	}
	body, _ := params["body"].(string)
	var labels []string
	if l, ok := params["labels"].([]interface{}); ok {
		for _, v := range l {
			if s, ok := v.(string); ok {
				labels = append(labels, s)
			}
		}
	}
	issue, err := a.ghClient().CreateIssue(ctx, github.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labels,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created issue #%d: %s\n%s", issue.Number, issue.Title, issue.URL), nil
}

// executeGitHubCommentIssue adds a comment to a GitHub issue.
// Params: number (required, int), body (required)
func (a *Agent) executeGitHubCommentIssue(ctx context.Context, params map[string]interface{}) (string, error) {
	number := 0
	switch v := params["number"].(type) {
	case float64:
		number = int(v)
	case int:
		number = v
	}
	if number == 0 {
		return "", fmt.Errorf("number parameter required")
	}
	body, _ := params["body"].(string)
	if body == "" {
		return "", fmt.Errorf("body parameter required")
	}
	if err := a.ghClient().CommentOnIssue(ctx, number, body); err != nil {
		return "", err
	}
	return fmt.Sprintf("Commented on issue #%d", number), nil
}

// executeGitHubCloseIssue closes a GitHub issue.
// Params: number (required, int), comment (optional)
func (a *Agent) executeGitHubCloseIssue(ctx context.Context, params map[string]interface{}) (string, error) {
	number := 0
	switch v := params["number"].(type) {
	case float64:
		number = int(v)
	case int:
		number = v
	}
	if number == 0 {
		return "", fmt.Errorf("number parameter required")
	}
	comment, _ := params["comment"].(string)
	if err := a.ghClient().CloseIssue(ctx, number, comment); err != nil {
		return "", err
	}
	return fmt.Sprintf("Closed issue #%d", number), nil
}

// executeGitHubListPRs lists pull requests.
// Params: state (optional: "open", "closed", "merged", "all")
func (a *Agent) executeGitHubListPRs(ctx context.Context, params map[string]interface{}) (string, error) {
	state, _ := params["state"].(string)
	prs, err := a.ghClient().ListPRs(ctx, state)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(prs, "", "  ")
	return string(out), nil
}

// executeGitHubWorkflowStatus lists recent workflow runs.
// Params: workflow (optional: filename like "ci.yml")
func (a *Agent) executeGitHubWorkflowStatus(ctx context.Context, params map[string]interface{}) (string, error) {
	workflow, _ := params["workflow"].(string)
	runs, err := a.ghClient().ListWorkflowRuns(ctx, workflow)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(runs, "", "  ")
	return string(out), nil
}
