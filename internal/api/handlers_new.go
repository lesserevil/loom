package api

import (
	"net/http"
)

func (s *Server) handleAgentPersona(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	agent, err := s.app.GetAgentManager().GetAgent(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Agent not found")
		return
	}

	if agent.Persona == nil {
		s.respondError(w, http.StatusNotFound, "Agent has no persona")
		return
	}

	s.respondJSON(w, http.StatusOK, agent.Persona)
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Context     string `json:"context"`
		ProjectID   string `json:"project_id"`
	}

	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" || req.Description == "" {
		s.respondError(w, http.StatusBadRequest, "title and description are required")
		return
	}

	if req.Type == "" {
		req.Type = "feedback"
	}

	if req.Type != "bug" && req.Type != "feature" && req.Type != "improvement" && req.Type != "feedback" {
		s.respondError(w, http.StatusBadRequest, "type must be bug, feature, improvement, or feedback")
		return
	}

	projectID := req.ProjectID
	if projectID == "" {
		projectID = s.defaultProjectID()
	}

	if projectID == "" {
		s.respondError(w, http.StatusBadRequest, "project_id is required or no default project configured")
		return
	}

	title := "[" + req.Type + "] " + req.Title
	description := req.Description
	if req.Context != "" {
		description = description + "\n\nContext: " + req.Context
	}

	bead, err := s.app.CreateBead(title, description, "P3", "feedback", projectID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, bead)
}
