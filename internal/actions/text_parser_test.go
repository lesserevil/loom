package actions

import (
	"testing"
)

func TestParseTextAction_Read(t *testing.T) {
	env, err := ParseTextAction("Let me read the config file.\n\nACTION: READ internal/config/config.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(env.Actions))
	}
	if env.Actions[0].Type != ActionReadFile {
		t.Errorf("expected read_file, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "internal/config/config.go" {
		t.Errorf("expected path internal/config/config.go, got %s", env.Actions[0].Path)
	}
}

func TestParseTextAction_Edit(t *testing.T) {
	response := `I'll fix the bug.

ACTION: EDIT internal/provider/registry.go
OLD:
<<<
func isHealthy(status string) bool {
	return status == "healthy"
}
>>>
NEW:
<<<
func isHealthy(status string) bool {
	return status == "healthy" || status == "active"
}
>>>`

	env, err := ParseTextAction(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(env.Actions))
	}
	if env.Actions[0].Type != ActionEditCode {
		t.Errorf("expected edit_code, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].OldText == "" {
		t.Error("OldText should not be empty")
	}
	if env.Actions[0].NewText == "" {
		t.Error("NewText should not be empty")
	}
}

func TestParseTextAction_Scope(t *testing.T) {
	env, err := ParseTextAction("ACTION: SCOPE internal/actions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionReadTree {
		t.Errorf("expected read_tree, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Path != "internal/actions" {
		t.Errorf("expected path internal/actions, got %s", env.Actions[0].Path)
	}
}

func TestParseTextAction_Done(t *testing.T) {
	env, err := ParseTextAction("All changes verified.\n\nACTION: DONE Fixed the provider activation bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionDone {
		t.Errorf("expected done, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestParseTextAction_MarkdownTolerance(t *testing.T) {
	// Model wraps in markdown list
	env, err := ParseTextAction("- ACTION: READ src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionReadFile {
		t.Errorf("expected read_file, got %s", env.Actions[0].Type)
	}
}

func TestParseTextAction_NoAction(t *testing.T) {
	_, err := ParseTextAction("I think we should review the code first and then decide.")
	if err == nil {
		t.Error("expected error for response with no action")
	}
}

func TestParseTextAction_Search(t *testing.T) {
	env, err := ParseTextAction("ACTION: SEARCH isProviderHealthy internal/provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionSearchText {
		t.Errorf("expected search_text, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Query != "isProviderHealthy" {
		t.Errorf("expected query isProviderHealthy, got %s", env.Actions[0].Query)
	}
	if env.Actions[0].Path != "internal/provider" {
		t.Errorf("expected path internal/provider, got %s", env.Actions[0].Path)
	}
}

func TestParseTextAction_Bash(t *testing.T) {
	env, err := ParseTextAction("ACTION: BASH go test ./internal/provider/...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Actions[0].Type != ActionRunCommand {
		t.Errorf("expected run_command, got %s", env.Actions[0].Type)
	}
	if env.Actions[0].Command != "go test ./internal/provider/..." {
		t.Errorf("unexpected command: %s", env.Actions[0].Command)
	}
}
