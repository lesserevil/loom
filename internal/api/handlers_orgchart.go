package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// handleOrgChart handles GET/PUT/POST /api/v1/org-charts/{projectId}
func (s *Server) handleOrgChart(w http.ResponseWriter, r *http.Request) {
	projectID := s.extractID(r.URL.Path, "/api/v1/org-charts")
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/org-charts/")
	parts := strings.Split(path, "/")

	switch r.Method {
	case http.MethodGet:
		chart, err := s.app.GetOrgChartManager().GetByProject(projectID)
		if err != nil {
			// If no org chart exists, create one from the project
			project, projErr := s.app.GetProjectManager().GetProject(projectID)
			if projErr != nil {
				s.respondError(w, http.StatusNotFound, "Project not found")
				return
			}
			chart, err = s.app.GetOrgChartManager().CreateForProject(projectID, project.Name)
			if err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		s.respondJSON(w, http.StatusOK, chart)

	case http.MethodPut:
		var req struct {
			Name      string            `json:"name"`
			Positions []models.Position `json:"positions"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		chart, err := s.app.GetOrgChartManager().GetByProject(projectID)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Org chart not found")
			return
		}

		if req.Name != "" {
			chart.Name = req.Name
		}
		if len(req.Positions) > 0 {
			chart.Positions = req.Positions
		}
		chart.UpdatedAt = time.Now()

		s.respondJSON(w, http.StatusOK, chart)

	case http.MethodPost:
		if len(parts) > 1 {
			action := parts[1]
			s.handleOrgChartAction(w, r, projectID, action)
			return
		}
		s.respondError(w, http.StatusBadRequest, "POST requires an action")

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleOrgChartAction handles POST actions on org charts
func (s *Server) handleOrgChartAction(w http.ResponseWriter, r *http.Request, projectID, action string) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chart, err := s.app.GetOrgChartManager().GetByProject(projectID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Org chart not found")
		return
	}

	switch action {
	case "positions":
		s.handleOrgChartPositions(w, r, projectID, chart)
	case "assign":
		s.handleOrgChartAssign(w, r, projectID, chart)
	case "unassign":
		s.handleOrgChartUnassign(w, r, projectID, chart)
	default:
		s.respondError(w, http.StatusBadRequest, "Unknown action")
	}
}

// handleOrgChartPositions handles adding/removing positions
func (s *Server) handleOrgChartPositions(w http.ResponseWriter, r *http.Request, projectID string, chart *models.OrgChart) {
	var req struct {
		Action   string         `json:"action"`
		Position models.Position `json:"position"`
		ID       string         `json:"id"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	switch req.Action {
	case "add":
		if err := s.app.GetOrgChartManager().AddPosition(projectID, req.Position); err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		chart, _ = s.app.GetOrgChartManager().GetByProject(projectID)
		s.respondJSON(w, http.StatusOK, chart)
	case "remove":
		if err := s.app.GetOrgChartManager().RemovePosition(projectID, req.ID); err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		chart, _ = s.app.GetOrgChartManager().GetByProject(projectID)
		s.respondJSON(w, http.StatusOK, chart)
	default:
		s.respondError(w, http.StatusBadRequest, "Unknown position action")
	}
}

// handleOrgChartAssign handles agent assignment to positions
func (s *Server) handleOrgChartAssign(w http.ResponseWriter, r *http.Request, projectID string, chart *models.OrgChart) {
	var req struct {
		PositionID string `json:"position_id"`
		AgentID    string `json:"agent_id"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PositionID == "" || req.AgentID == "" {
		s.respondError(w, http.StatusBadRequest, "position_id and agent_id are required")
		return
	}

	if err := s.app.GetOrgChartManager().AssignAgent(projectID, req.PositionID, req.AgentID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	chart, _ = s.app.GetOrgChartManager().GetByProject(projectID)
	s.respondJSON(w, http.StatusOK, chart)
}

// handleOrgChartUnassign handles agent removal from positions
func (s *Server) handleOrgChartUnassign(w http.ResponseWriter, r *http.Request, projectID string, chart *models.OrgChart) {
	var req struct {
		PositionID string `json:"position_id"`
		AgentID    string `json:"agent_id"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PositionID == "" || req.AgentID == "" {
		s.respondError(w, http.StatusBadRequest, "position_id and agent_id are required")
		return
	}

	if err := s.app.GetOrgChartManager().UnassignAgent(projectID, req.PositionID, req.AgentID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	chart, _ = s.app.GetOrgChartManager().GetByProject(projectID)
	s.respondJSON(w, http.StatusOK, chart)
}
