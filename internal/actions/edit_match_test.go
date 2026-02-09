package actions

import (
	"testing"
)

func TestMatchAndReplace_Exact(t *testing.T) {
	content := "func hello() {\n\treturn \"world\"\n}"
	result, ok, strategy := MatchAndReplace(content, "return \"world\"", "return \"universe\"")
	if !ok {
		t.Fatal("expected match")
	}
	if strategy != "exact" {
		t.Errorf("expected exact strategy, got %s", strategy)
	}
	if result != "func hello() {\n\treturn \"universe\"\n}" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestMatchAndReplace_LineTrimmed(t *testing.T) {
	// Content has trailing spaces, old text doesn't
	content := "func foo() {  \n\tbar()  \n}  "
	result, ok, strategy := MatchAndReplace(content, "func foo() {\n\tbar()\n}", "func baz() {\n\tqux()\n}")
	if !ok {
		t.Fatal("expected match")
	}
	if strategy != "line_trimmed" {
		t.Errorf("expected line_trimmed, got %s", strategy)
	}
	_ = result
}

func TestMatchAndReplace_IndentFlexible(t *testing.T) {
	// Content is indented with tabs, old text uses spaces.
	// whitespace_normalized or indentation_flexible may match — both are correct.
	content := "\tfunc isHealthy() bool {\n\t\treturn true\n\t}"
	_, ok, strategy := MatchAndReplace(content, "func isHealthy() bool {\n    return true\n}", "func isHealthy() bool {\n    return false\n}")
	if !ok {
		t.Fatal("expected match")
	}
	if strategy != "whitespace_normalized" && strategy != "indentation_flexible" {
		t.Errorf("expected whitespace_normalized or indentation_flexible, got %s", strategy)
	}
}

func TestMatchAndReplace_BlockAnchor(t *testing.T) {
	content := "package main\n\nfunc alpha() {\n\tx := 1\n\ty := 2\n\tz := 3\n}\n\nfunc beta() {}"
	// Old text has the right first/last lines but slightly different middle
	oldText := "func alpha() {\n\tx := 1\n\tz := 3\n}"
	newText := "func alpha() {\n\tx := 10\n}"
	result, ok, strategy := MatchAndReplace(content, oldText, newText)
	if !ok {
		t.Fatal("expected match")
	}
	if strategy != "block_anchor" {
		t.Errorf("expected block_anchor, got %s", strategy)
	}
	_ = result
}

func TestMatchAndReplace_NoMatch(t *testing.T) {
	content := "func hello() {}"
	_, ok, _ := MatchAndReplace(content, "this text does not exist anywhere", "replacement")
	if ok {
		t.Fatal("expected no match")
	}
}
