package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleConversationsList_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations?project_id=loom&limit=50", nil)
	w := httptest.NewRecorder()
	s.handleConversationsList(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConversationsList_BadRequest(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?limit=50", nil)
	w := httptest.NewRecorder()
	s.handleConversationsList(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleConversationsList_Success(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?project_id=loom&limit=50", nil)
	w := httptest.NewRecorder()
	s.handleConversationsList(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
