package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jordanhubbard/loom/internal/memory"
)

// handleProjectMemory dispatches /api/v1/projects/{id}/memory[/...] requests.
//
//	GET    /api/v1/projects/{id}/memory              — all entries
//	GET    /api/v1/projects/{id}/memory/{category}   — entries by category
//	PUT    /api/v1/projects/{id}/memory/{cat}/{key}  — upsert an entry
//	DELETE /api/v1/projects/{id}/memory/{cat}/{key}  — delete an entry
func (s *Server) handleProjectMemory(w http.ResponseWriter, r *http.Request, projectID string) {
	mm := s.app.GetMemoryManager()
	if mm == nil {
		s.respondError(w, http.StatusServiceUnavailable, "memory manager not available (no database)")
		return
	}

	prefix := fmt.Sprintf("/api/v1/projects/%s/memory", projectID)
	sub := strings.TrimPrefix(r.URL.Path, prefix)
	sub = strings.TrimPrefix(sub, "/")
	parts := strings.SplitN(sub, "/", 2)

	switch {
	case sub == "":
		// GET /memory — all entries
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		entries, err := mm.All(r.Context(), projectID)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, entries)

	case len(parts) == 1:
		// GET /memory/{category}
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cat := memory.MemoryCategory(parts[0])
		entries, err := mm.GetByCategory(r.Context(), projectID, cat)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, entries)

	case len(parts) == 2:
		cat := memory.MemoryCategory(parts[0])
		key := parts[1]
		switch r.Method {
		case http.MethodPut:
			var req struct {
				Value      string  `json:"value"`
				Confidence float64 `json:"confidence"`
				SourceBead string  `json:"source_bead"`
			}
			if err := s.parseJSON(r, &req); err != nil {
				s.respondError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if req.Value == "" {
				s.respondError(w, http.StatusBadRequest, "value required")
				return
			}
			if req.Confidence == 0 {
				req.Confidence = 1.0
			}
			entry := &memory.ProjectMemory{
				ProjectID:  projectID,
				Category:   cat,
				Key:        key,
				Value:      req.Value,
				Confidence: req.Confidence,
				SourceBead: req.SourceBead,
			}
			if err := mm.Set(r.Context(), entry); err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.respondJSON(w, http.StatusOK, entry)

		case http.MethodDelete:
			if err := mm.Delete(r.Context(), projectID, cat, key); err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

		default:
			s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		}

	default:
		s.respondError(w, http.StatusNotFound, "unknown memory endpoint")
	}
}
