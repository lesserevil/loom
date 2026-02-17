package actions

import (
	"context"
	"errors"
	"testing"
)

// mockMCPToolCaller implements MCPToolCaller for testing
type mockMCPToolCaller struct {
	calls  []toolCall
	result map[string]interface{}
	err    error
}

type toolCall struct {
	toolName string
	params   map[string]interface{}
}

func (m *mockMCPToolCaller) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	m.calls = append(m.calls, toolCall{toolName: toolName, params: params})
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestWorkflowMCPAdapter_StartDevelopment(t *testing.T) {
	tests := []struct {
		name           string
		workflow       string
		requireReviews bool
		projectPath    string
		mockResult     map[string]interface{}
		mockError      error
		wantError      bool
		wantToolName   string
	}{
		{
			name:           "successful start with all params",
			workflow:       "epcc",
			requireReviews: true,
			projectPath:    "/test/project",
			mockResult:     map[string]interface{}{"instructions": "workflow started"},
			wantToolName:   "mcp__responsible-vibe-mcp__start_development",
		},
		{
			name:           "successful start without project path",
			workflow:       "tdd",
			requireReviews: false,
			projectPath:    "",
			mockResult:     map[string]interface{}{"success": true},
			wantToolName:   "mcp__responsible-vibe-mcp__start_development",
		},
		{
			name:         "mcp tool error",
			workflow:     "waterfall",
			mockError:    errors.New("MCP tool failed"),
			wantError:    true,
			wantToolName: "mcp__responsible-vibe-mcp__start_development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockMCPToolCaller{
				result: tt.mockResult,
				err:    tt.mockError,
			}
			adapter := NewWorkflowMCPAdapter(mock)

			result, err := adapter.StartDevelopment(context.Background(), tt.workflow, tt.requireReviews, tt.projectPath)

			if (err != nil) != tt.wantError {
				t.Errorf("StartDevelopment() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if result == nil {
					t.Error("StartDevelopment() result is nil")
					return
				}
			}

			// Verify tool was called with correct name
			if len(mock.calls) != 1 {
				t.Errorf("expected 1 tool call, got %d", len(mock.calls))
				return
			}
			if mock.calls[0].toolName != tt.wantToolName {
				t.Errorf("tool name = %v, want %v", mock.calls[0].toolName, tt.wantToolName)
			}

			// Verify parameters
			params := mock.calls[0].params
			if params["workflow"] != tt.workflow {
				t.Errorf("workflow param = %v, want %v", params["workflow"], tt.workflow)
			}
			if params["require_reviews"] != tt.requireReviews {
				t.Errorf("require_reviews param = %v, want %v", params["require_reviews"], tt.requireReviews)
			}
			if tt.projectPath != "" && params["project_path"] != tt.projectPath {
				t.Errorf("project_path param = %v, want %v", params["project_path"], tt.projectPath)
			}
		})
	}
}

func TestWorkflowMCPAdapter_WhatsNext(t *testing.T) {
	mock := &mockMCPToolCaller{
		result: map[string]interface{}{"instructions": "next steps"},
	}
	adapter := NewWorkflowMCPAdapter(mock)

	recentMessages := []map[string]string{
		{"role": "user", "content": "hello"},
	}

	result, err := adapter.WhatsNext(context.Background(), "user input", "context", "summary", recentMessages)

	if err != nil {
		t.Errorf("WhatsNext() unexpected error = %v", err)
	}
	if result == nil {
		t.Error("WhatsNext() result is nil")
	}

	// Verify tool called
	if len(mock.calls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(mock.calls))
	}
	if mock.calls[0].toolName != "mcp__responsible-vibe-mcp__whats_next" {
		t.Errorf("tool name = %v, want mcp__responsible-vibe-mcp__whats_next", mock.calls[0].toolName)
	}
}

func TestWorkflowMCPAdapter_ProceedToPhase(t *testing.T) {
	mock := &mockMCPToolCaller{
		result: map[string]interface{}{"phase": "implementation"},
	}
	adapter := NewWorkflowMCPAdapter(mock)

	result, err := adapter.ProceedToPhase(context.Background(), "implementation", "performed", "ready to code")

	if err != nil {
		t.Errorf("ProceedToPhase() unexpected error = %v", err)
	}
	if result == nil {
		t.Error("ProceedToPhase() result is nil")
	}

	// Verify parameters
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(mock.calls))
	}
	params := mock.calls[0].params
	if params["target_phase"] != "implementation" {
		t.Errorf("target_phase = %v, want implementation", params["target_phase"])
	}
	if params["review_state"] != "performed" {
		t.Errorf("review_state = %v, want performed", params["review_state"])
	}
	if params["reason"] != "ready to code" {
		t.Errorf("reason = %v, want 'ready to code'", params["reason"])
	}
}

func TestWorkflowMCPAdapter_ConductReview(t *testing.T) {
	mock := &mockMCPToolCaller{
		result: map[string]interface{}{"review_complete": true},
	}
	adapter := NewWorkflowMCPAdapter(mock)

	result, err := adapter.ConductReview(context.Background(), "design")

	if err != nil {
		t.Errorf("ConductReview() unexpected error = %v", err)
	}
	if result == nil {
		t.Error("ConductReview() result is nil")
	}

	// Verify tool called with correct target phase
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(mock.calls))
	}
	if mock.calls[0].params["target_phase"] != "design" {
		t.Errorf("target_phase = %v, want design", mock.calls[0].params["target_phase"])
	}
}

func TestWorkflowMCPAdapter_ResumeWorkflow(t *testing.T) {
	tests := []struct {
		name                string
		includeSystemPrompt bool
	}{
		{"with system prompt", true},
		{"without system prompt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockMCPToolCaller{
				result: map[string]interface{}{"resumed": true},
			}
			adapter := NewWorkflowMCPAdapter(mock)

			result, err := adapter.ResumeWorkflow(context.Background(), tt.includeSystemPrompt)

			if err != nil {
				t.Errorf("ResumeWorkflow() unexpected error = %v", err)
			}
			if result == nil {
				t.Error("ResumeWorkflow() result is nil")
			}

			// Verify parameter
			if len(mock.calls) != 1 {
				t.Fatalf("expected 1 tool call, got %d", len(mock.calls))
			}
			if mock.calls[0].params["include_system_prompt"] != tt.includeSystemPrompt {
				t.Errorf("include_system_prompt = %v, want %v", mock.calls[0].params["include_system_prompt"], tt.includeSystemPrompt)
			}
		})
	}
}

func TestWorkflowMCPAdapter_ErrorHandling(t *testing.T) {
	testError := errors.New("MCP connection failed")
	mock := &mockMCPToolCaller{
		err: testError,
	}
	adapter := NewWorkflowMCPAdapter(mock)

	// Test each method returns wrapped error
	_, err := adapter.StartDevelopment(context.Background(), "epcc", false, "")
	if err == nil || !errors.Is(err, testError) {
		t.Error("StartDevelopment should return wrapped error")
	}

	_, err = adapter.WhatsNext(context.Background(), "", "", "", nil)
	if err == nil || !errors.Is(err, testError) {
		t.Error("WhatsNext should return wrapped error")
	}

	_, err = adapter.ProceedToPhase(context.Background(), "phase", "state", "")
	if err == nil || !errors.Is(err, testError) {
		t.Error("ProceedToPhase should return wrapped error")
	}

	_, err = adapter.ConductReview(context.Background(), "phase")
	if err == nil || !errors.Is(err, testError) {
		t.Error("ConductReview should return wrapped error")
	}

	_, err = adapter.ResumeWorkflow(context.Background(), false)
	if err == nil || !errors.Is(err, testError) {
		t.Error("ResumeWorkflow should return wrapped error")
	}
}
