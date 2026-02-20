package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jordanhubbard/loom/internal/github"
)

// handleProjectGitHub dispatches /api/v1/projects/{id}/github[/...] requests.
//
//	PUT  /api/v1/projects/{id}/github           — set github_repo
//	GET  /api/v1/projects/{id}/github           — get github info
//	GET  /api/v1/projects/{id}/github/issues    — list issues
//	POST /api/v1/projects/{id}/github/issues    — create issue
//	GET  /api/v1/projects/{id}/github/prs       — list PRs
//	POST /api/v1/projects/{id}/github/prs       — create PR
//	GET  /api/v1/projects/{id}/github/actions   — list workflow runs
func (s *Server) handleProjectGitHub(w http.ResponseWriter, r *http.Request, projectID string) {
	// Parse sub-path after /github
	prefix := fmt.Sprintf("/api/v1/projects/%s/github", projectID)
	sub := strings.TrimPrefix(r.URL.Path, prefix)
	sub = strings.TrimPrefix(sub, "/")

	switch {
	case sub == "" || sub == "/":
		s.handleGitHubRoot(w, r, projectID)
	case sub == "issues":
		s.handleGitHubIssues(w, r, projectID)
	case sub == "prs":
		s.handleGitHubPRs(w, r, projectID)
	case sub == "actions":
		s.handleGitHubActions(w, r, projectID)
	default:
		s.respondError(w, http.StatusNotFound, "unknown github endpoint")
	}
}

// githubClientForProject creates a GitHub client for the project's workspace.
func (s *Server) githubClientForProject(projectID string) (*github.Client, error) {
	pm := s.app.GetProjectManager()
	project, err := pm.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Use project WorkDir if set, otherwise derive from data directory convention.
	workDir := project.WorkDir
	if workDir == "" {
		workDir = fmt.Sprintf("data/projects/%s/main", projectID)
	}

	// GH_TOKEN can be stored in project context or as env var; we pass empty
	// to let gh use its stored credentials.
	token := project.Context["github_token"]
	return github.NewClient(workDir, token), nil
}

func (s *Server) handleGitHubRoot(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodGet:
		pm := s.app.GetProjectManager()
		project, err := pm.GetProject(projectID)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "project not found")
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"github_repo":    project.GitHubRepo,
			"default_branch": project.DefaultBranch,
		})

	case http.MethodPut:
		var req struct {
			GitHubRepo    string `json:"github_repo"`
			DefaultBranch string `json:"default_branch"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updates := map[string]interface{}{}
		if req.GitHubRepo != "" {
			updates["github_repo"] = req.GitHubRepo
		}
		if req.DefaultBranch != "" {
			updates["default_branch"] = req.DefaultBranch
		}
		if len(updates) == 0 {
			s.respondError(w, http.StatusBadRequest, "github_repo or default_branch required")
			return
		}
		if err := s.app.GetProjectManager().UpdateProject(projectID, updates); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.app.PersistProject(projectID)
		project, _ := s.app.GetProjectManager().GetProject(projectID)
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"github_repo":    project.GitHubRepo,
			"default_branch": project.DefaultBranch,
		})

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGitHubIssues(w http.ResponseWriter, r *http.Request, projectID string) {
	client, err := s.githubClientForProject(projectID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		state := r.URL.Query().Get("state")
		issues, err := client.ListIssues(r.Context(), state)
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, issues)

	case http.MethodPost:
		var req struct {
			Title  string   `json:"title"`
			Body   string   `json:"body"`
			Labels []string `json:"labels"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Title == "" {
			s.respondError(w, http.StatusBadRequest, "title required")
			return
		}
		issue, err := client.CreateIssue(r.Context(), github.CreateIssueRequest{
			Title:  req.Title,
			Body:   req.Body,
			Labels: req.Labels,
		})
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusCreated, issue)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGitHubPRs(w http.ResponseWriter, r *http.Request, projectID string) {
	client, err := s.githubClientForProject(projectID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		state := r.URL.Query().Get("state")
		prs, err := client.ListPRs(r.Context(), state)
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, prs)

	case http.MethodPost:
		var req struct {
			Title string `json:"title"`
			Body  string `json:"body"`
			Head  string `json:"head"`
			Base  string `json:"base"`
			Draft bool   `json:"draft"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Title == "" {
			s.respondError(w, http.StatusBadRequest, "title required")
			return
		}
		pr, err := client.CreatePR(r.Context(), github.CreatePRRequest{
			Title: req.Title,
			Body:  req.Body,
			Head:  req.Head,
			Base:  req.Base,
			Draft: req.Draft,
		})
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusCreated, pr)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGitHubActions(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client, err := s.githubClientForProject(projectID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	workflow := r.URL.Query().Get("workflow")
	runs, err := client.ListWorkflowRuns(r.Context(), workflow)
	if err != nil {
		s.respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, runs)
}
