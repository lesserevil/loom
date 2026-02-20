package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleConnectors_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/connectors", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConnectors_HealthMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connectors/health", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConnectors_SubPath_TestMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/conn1/test", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConnectors_SubPath_HealthMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connectors/conn1/health", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleConnectors_SubPath_Unknown(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connectors/conn1/unknown", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleConnectors_IdMethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/connectors/conn1", nil)
	w := httptest.NewRecorder()
	s.HandleConnectors(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
