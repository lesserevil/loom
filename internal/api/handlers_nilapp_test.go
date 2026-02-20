package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGetChangeVelocity_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/change-velocity", nil)
	w := httptest.NewRecorder()
	s.handleGetChangeVelocity(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleGetChangeVelocity_MissingProjectID(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/change-velocity", nil)
	w := httptest.NewRecorder()
	s.handleGetChangeVelocity(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleGetChangeVelocity_BadTimeWindow(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/change-velocity?project_id=p1&time_window=notvalid", nil)
	w := httptest.NewRecorder()
	s.handleGetChangeVelocity(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCloneAgent_BadJSON(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a1/clone", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	s.handleCloneAgent(w, req, "a1")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
