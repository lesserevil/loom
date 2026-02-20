package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDocs_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", nil)
	w := httptest.NewRecorder()
	s.handleDocs(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleDocs_PathTraversal(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs?path=../../etc/passwd", nil)
	w := httptest.NewRecorder()
	s.handleDocs(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal, got %d", w.Code)
	}
}

func TestHandleDocs_PageNotFound(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs?path=nonexistent.md", nil)
	w := httptest.NewRecorder()
	s.handleDocs(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExtractTitleFromContent(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{"# Hello World\n\nSome content", "Hello World"},
		{"No heading here", "Untitled"},
		{"## Sub Heading\n# Main Heading", "Main Heading"},
		{"", "Untitled"},
	}
	for _, tt := range tests {
		got := extractTitleFromContent(tt.content)
		if got != tt.want {
			t.Errorf("extractTitle(%q) = %q, want %q", tt.content[:min(len(tt.content), 30)], got, tt.want)
		}
	}
}

func TestClassifySection(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"getting-started/quickstart.md", "Getting Started"},
		{"guide/user/dashboard.md", "User Guide"},
		{"guide/admin/config.md", "Administrator Guide"},
		{"guide/developer/api.md", "Developer Guide"},
		{"guide/reference/entities.md", "Reference"},
		{"guide/tutorials/story.md", "Tutorials"},
		{"ARCHITECTURE.md", "General"},
	}
	for _, tt := range tests {
		got := classifySection(tt.path)
		if got != tt.want {
			t.Errorf("classifySection(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestHandleDocs_Index(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	w := httptest.NewRecorder()
	s.handleDocs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// count key should always be present (0 if docs dir doesn't exist)
	if _, ok := result["count"]; !ok {
		t.Error("expected 'count' key in response")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
