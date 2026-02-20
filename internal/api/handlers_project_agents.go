package api

import (
	"log"
	"net/http"
	"strings"
)

// handleProjectAgentRegister handles POST /api/v1/project-agents/register
// Called by project agent containers when they start up.
func (s *Server) handleProjectAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ProjectID string   `json:"project_id"`
		WorkDir   string   `json:"work_dir"`
		AgentURL  string   `json:"agent_url"`
		Roles     []string `json:"roles,omitempty"`
	}
	if err := s.parseJSON(r, &payload); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.ProjectID == "" || payload.AgentURL == "" {
		s.respondError(w, http.StatusBadRequest, "project_id and agent_url are required")
		return
	}

	orch := s.app.GetContainerOrchestrator()
	if orch == nil {
		s.respondError(w, http.StatusServiceUnavailable, "container orchestrator not available")
		return
	}

	orch.RegisterAgent(payload.ProjectID, payload.AgentURL, payload.Roles)
	log.Printf("[API] Project agent registered: project=%s url=%s roles=%v", payload.ProjectID, payload.AgentURL, payload.Roles)

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// handleProjectAgentHeartbeat handles POST /api/v1/project-agents/{id}/heartbeat
func (s *Server) handleProjectAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Just acknowledge - we don't track heartbeat timestamps yet
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleProjectAgentResults handles POST /api/v1/project-agents/{id}/results
func (s *Server) handleProjectAgentResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Results logged for now; future: route to waiting callers
	var result map[string]interface{}
	if err := s.parseJSON(r, &result); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid result body")
		return
	}

	projectID := extractProjectAgentID(r.URL.Path)
	log.Printf("[API] Task result received from project agent %s: %v", projectID, result)
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

// handleContainerAgents dispatches /api/v1/project-agents/* requests
func (s *Server) handleContainerAgents(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/project-agents")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "register":
		s.handleProjectAgentRegister(w, r)
	case strings.HasSuffix(path, "/heartbeat"):
		s.handleProjectAgentHeartbeat(w, r)
	case strings.HasSuffix(path, "/results"):
		s.handleProjectAgentResults(w, r)
	default:
		http.NotFound(w, r)
	}
}

func extractProjectAgentID(urlPath string) string {
	// /api/v1/project-agents/{id}/heartbeat → id
	path := strings.TrimPrefix(urlPath, "/api/v1/project-agents/")
	parts := strings.SplitN(path, "/", 2)
	return parts[0]
}
