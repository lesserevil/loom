package projectagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteCreatePR_MissingTitle(t *testing.T) {
	agent := newTestAgent(t)
	_, err := agent.executeCreatePR(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestExecuteCreateBead_MissingTitle(t *testing.T) {
	agent := newTestAgent(t)
	_, err := agent.executeCreateBead(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestExecuteCreateBead_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/beads" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Test Bug" {
			t.Errorf("expected title 'Test Bug', got %v", body["title"])
		}
		if body["type"] != "decision" {
			t.Errorf("expected type 'decision', got %v", body["type"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "bead-123"})
	}))
	defer srv.Close()

	agent, _ := New(Config{
		ProjectID:       "test-proj",
		ControlPlaneURL: srv.URL,
		WorkDir:         t.TempDir(),
	})

	output, err := agent.executeCreateBead(context.Background(), map[string]interface{}{
		"title":       "Test Bug",
		"description": "A test bug",
		"type":        "decision",
		"priority":    float64(0),
		"tags":        []interface{}{"test", "approval"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestExecuteCreateBead_ControlPlaneError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "db down"})
	}))
	defer srv.Close()

	agent, _ := New(Config{
		ProjectID:       "test-proj",
		ControlPlaneURL: srv.URL,
		WorkDir:         t.TempDir(),
	})

	_, err := agent.executeCreateBead(context.Background(), map[string]interface{}{
		"title": "Will Fail",
	})
	if err == nil {
		t.Error("expected error from control plane 500")
	}
}

func TestExecuteCloseBead_MissingParams(t *testing.T) {
	agent := newTestAgent(t)
	_, err := agent.executeCloseBead(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing bead_id")
	}

	_, err = agent.executeCloseBead(context.Background(), map[string]interface{}{
		"bead_id": "b1",
	})
	if err == nil {
		t.Error("expected error for missing reason")
	}
}

func TestExecuteCloseBead_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/beads/bead-99" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	agent, _ := New(Config{
		ProjectID:       "test-proj",
		ControlPlaneURL: srv.URL,
		WorkDir:         t.TempDir(),
	})

	output, err := agent.executeCloseBead(context.Background(), map[string]interface{}{
		"bead_id": "bead-99",
		"reason":  "Fixed and verified",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestExecuteVerify_AutoDetect(t *testing.T) {
	agent := newTestAgent(t)
	// Create a go.mod so the auto-detect picks "go test"
	os.WriteFile(filepath.Join(agent.config.WorkDir, "go.mod"), []byte("module test"), 0644)

	cmd := agent.detectTestCommand()
	if cmd != "go test ./... 2>&1" {
		t.Errorf("expected 'go test ./... 2>&1', got %q", cmd)
	}
}

func TestExecuteVerify_CustomCommand(t *testing.T) {
	agent := newTestAgent(t)
	output, err := agent.executeVerify(context.Background(), map[string]interface{}{
		"command": "echo all_tests_pass",
		"timeout": float64(5),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestDetectTestCommand_NoFramework(t *testing.T) {
	agent := newTestAgent(t)
	cmd := agent.detectTestCommand()
	if cmd != "echo 'No test framework detected'" {
		t.Errorf("expected fallback, got %q", cmd)
	}
}

func TestDetectTestCommand_Makefile(t *testing.T) {
	agent := newTestAgent(t)
	os.WriteFile(filepath.Join(agent.config.WorkDir, "Makefile"), []byte("test:\n\techo ok"), 0644)
	cmd := agent.detectTestCommand()
	if cmd != "make test 2>&1" {
		t.Errorf("expected 'make test 2>&1', got %q", cmd)
	}
}

func TestExecuteAction_NewTypes(t *testing.T) {
	agent := newTestAgent(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		action   LLMAction
		wantType string
	}{
		{
			name:     "create_pr no title",
			action:   LLMAction{Type: "create_pr", Params: map[string]interface{}{}},
			wantType: "create_pr",
		},
		{
			name:     "create_bead no title",
			action:   LLMAction{Type: "create_bead", Params: map[string]interface{}{}},
			wantType: "create_bead",
		},
		{
			name:     "close_bead no params",
			action:   LLMAction{Type: "close_bead", Params: map[string]interface{}{}},
			wantType: "close_bead",
		},
		{
			name:     "verify with echo",
			action:   LLMAction{Type: "verify", Params: map[string]interface{}{"command": "echo pass"}},
			wantType: "verify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.executeAction(ctx, tt.action)
			if result.Type != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, result.Type)
			}
		})
	}
}
