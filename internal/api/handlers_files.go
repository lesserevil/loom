package api

import (
	"net/http"
	"strconv"
	"strings"
)

// handleProjectFiles handles /api/v1/projects/{id}/files/*
func (s *Server) handleProjectFiles(w http.ResponseWriter, r *http.Request, projectID string, parts []string) {
	if s.fileManager == nil {
		s.respondError(w, http.StatusInternalServerError, "file manager not configured")
		return
	}
	if len(parts) == 0 || parts[0] == "" {
		s.respondError(w, http.StatusNotFound, "file action required")
		return
	}

	action := parts[0]
	switch action {
	case "read":
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		path := r.URL.Query().Get("path")
		if path == "" {
			s.respondError(w, http.StatusBadRequest, "path is required")
			return
		}
		res, err := s.fileManager.ReadFile(r.Context(), projectID, path)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"path":    res.Path,
			"content": res.Content,
			"size":    res.Size,
		})
	case "tree":
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		path := r.URL.Query().Get("path")
		maxDepth := parseInt(r.URL.Query().Get("max_depth"))
		limit := parseInt(r.URL.Query().Get("limit"))
		res, err := s.fileManager.ReadTree(r.Context(), projectID, path, maxDepth, limit)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"entries": res,
		})
	case "search":
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		path := r.URL.Query().Get("path")
		query := r.URL.Query().Get("query")
		if strings.TrimSpace(query) == "" {
			s.respondError(w, http.StatusBadRequest, "query is required")
			return
		}
		limit := parseInt(r.URL.Query().Get("limit"))
		res, err := s.fileManager.SearchText(r.Context(), projectID, path, query, limit)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"matches": res,
		})
	case "patch":
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var req struct {
			Patch string `json:"patch"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if strings.TrimSpace(req.Patch) == "" {
			s.respondError(w, http.StatusBadRequest, "patch is required")
			return
		}
		res, err := s.fileManager.ApplyPatch(r.Context(), projectID, req.Patch)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"applied": res.Applied,
			"output":  res.Output,
		})
	default:
		s.respondError(w, http.StatusNotFound, "Unknown file action")
	}
}

func parseInt(value string) int {
	if value == "" {
		return 0
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return 0
}
