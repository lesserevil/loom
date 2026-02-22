package audit

import (
	"testing"
)

func TestParser_ParseGoBuild_FileLineCol(t *testing.T) {
	parser := NewParser()
	output := `/home/jkh/Src/loom/internal/foo.go:42:10: undefined: Bar
/home/jkh/Src/loom/internal/baz.go:7:3: cannot use x (type int) as type string`

	findings := parser.ParseGoBuild(output)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	f := findings[0]
	if f.Type != FindingTypeBuildError {
		t.Errorf("type = %q, want %q", f.Type, FindingTypeBuildError)
	}
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want %q", f.Severity, SeverityError)
	}
	if f.Source != "go build" {
		t.Errorf("source = %q, want %q", f.Source, "go build")
	}
	if f.File != "/home/jkh/Src/loom/internal/foo.go" {
		t.Errorf("file = %q, want %q", f.File, "/home/jkh/Src/loom/internal/foo.go")
	}
	if f.Line != 42 {
		t.Errorf("line = %d, want 42", f.Line)
	}
	if f.Column != 10 {
		t.Errorf("column = %d, want 10", f.Column)
	}
	if f.Message != "undefined: Bar" {
		t.Errorf("message = %q, want %q", f.Message, "undefined: Bar")
	}
}

func TestParser_ParseGoBuild_SkipsComments(t *testing.T) {
	parser := NewParser()
	output := `# github.com/jordanhubbard/loom/internal/foo
internal/foo.go:5:2: undefined: X`

	findings := parser.ParseGoBuild(output)

	// Should skip the "# ..." line and parse the error
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].File != "internal/foo.go" {
		t.Errorf("file = %q, want %q", findings[0].File, "internal/foo.go")
	}
}

func TestParser_ParseGoBuild_CatchAll(t *testing.T) {
	parser := NewParser()
	output := `no package found in any of: /usr/local/go/src/nonexistent`

	findings := parser.ParseGoBuild(output)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].File != "" {
		t.Errorf("expected empty file for catch-all, got %q", findings[0].File)
	}
}

func TestParser_ParseGoBuild_Empty(t *testing.T) {
	parser := NewParser()
	findings := parser.ParseGoBuild("")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty output, got %d", len(findings))
	}
}

func TestParser_ParseGoTest_FailedTest(t *testing.T) {
	parser := NewParser()
	output := `--- FAIL: TestSomething (0.01s)
    foo_test.go:15: expected 3, got 5
FAIL
FAIL	github.com/jordanhubbard/loom/internal/foo	0.012s`

	findings := parser.ParseGoTest(output)

	// Should get the file-level finding + the FAIL lines
	found := false
	for _, f := range findings {
		if f.Rule == "TestSomething" {
			found = true
			if f.Type != FindingTypeTestFailure {
				t.Errorf("type = %q, want %q", f.Type, FindingTypeTestFailure)
			}
			if f.File != "foo_test.go" {
				t.Errorf("file = %q, want %q", f.File, "foo_test.go")
			}
			if f.Line != 15 {
				t.Errorf("line = %d, want 15", f.Line)
			}
		}
	}
	if !found {
		t.Error("did not find TestSomething failure in findings")
	}
}

func TestParser_ParseGoTest_Panic(t *testing.T) {
	parser := NewParser()
	output := `panic: runtime error: index out of range [5] with length 3`

	findings := parser.ParseGoTest(output)

	if len(findings) == 0 {
		t.Fatal("expected at least 1 finding for panic")
	}

	found := false
	for _, f := range findings {
		if f.Type == FindingTypeTestFailure && f.Source == "go test" {
			found = true
		}
	}
	if !found {
		t.Error("did not find panic finding")
	}
}

func TestParser_ParseGoTest_AllPassing(t *testing.T) {
	parser := NewParser()
	output := `ok  	github.com/jordanhubbard/loom/internal/foo	0.123s
ok  	github.com/jordanhubbard/loom/internal/bar	0.045s
PASS`

	findings := parser.ParseGoTest(output)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for passing tests, got %d", len(findings))
	}
}

func TestParser_ParseGoLint_PlainFormat(t *testing.T) {
	parser := NewParser()
	output := `internal/foo.go:10:5: exported function Foo should have comment or be unexported (golint)
internal/bar.go:20:1: error return value not checked (errcheck)`

	findings := parser.ParseGoLint(output)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	f := findings[0]
	if f.Type != FindingTypeLintError {
		t.Errorf("type = %q, want %q", f.Type, FindingTypeLintError)
	}
	if f.Source != "golangci-lint" {
		t.Errorf("source = %q, want %q", f.Source, "golangci-lint")
	}
	if f.File != "internal/foo.go" {
		t.Errorf("file = %q, want %q", f.File, "internal/foo.go")
	}
	if f.Line != 10 {
		t.Errorf("line = %d, want 10", f.Line)
	}
}

func TestParser_ParseGoLint_SkipsSummary(t *testing.T) {
	parser := NewParser()
	output := `== linting complete ==
-- summary --`

	findings := parser.ParseGoLint(output)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for summary lines, got %d", len(findings))
	}
}

func TestParser_ParseGoLint_Empty(t *testing.T) {
	parser := NewParser()
	findings := parser.ParseGoLint("")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty output, got %d", len(findings))
	}
}

func TestParser_Parse_Dispatch(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		source   string
		output   string
		wantLen  int
		wantType FindingType
	}{
		{"go build", "foo.go:1:1: err", 1, FindingTypeBuildError},
		{"go test", "--- FAIL: TestX\n    x_test.go:1: bad\nFAIL", 2, FindingTypeTestFailure},
		{"golangci-lint", "foo.go:1:1: msg (rule)", 1, FindingTypeLintError},
		{"unknown", "anything", 0, ""},
	}

	for _, tt := range tests {
		findings := parser.Parse(tt.source, tt.output)
		if len(findings) < tt.wantLen {
			t.Errorf("Parse(%q): got %d findings, want at least %d", tt.source, len(findings), tt.wantLen)
		}
		if tt.wantLen > 0 && findings[0].Type != tt.wantType {
			t.Errorf("Parse(%q): type = %q, want %q", tt.source, findings[0].Type, tt.wantType)
		}
	}
}

func TestParser_NewResult_AllPassing(t *testing.T) {
	parser := NewParser()
	result := parser.NewResult(nil)

	if !result.BuildPassed {
		t.Error("expected BuildPassed = true")
	}
	if !result.TestPassed {
		t.Error("expected TestPassed = true")
	}
	if !result.LintPassed {
		t.Error("expected LintPassed = true")
	}
	if result.Summary != "All checks passed" {
		t.Errorf("summary = %q, want %q", result.Summary, "All checks passed")
	}
}

func TestParser_NewResult_Mixed(t *testing.T) {
	parser := NewParser()
	findings := []Finding{
		{Type: FindingTypeBuildError, Severity: SeverityError},
		{Type: FindingTypeBuildError, Severity: SeverityError},
		{Type: FindingTypeTestFailure, Severity: SeverityError},
		{Type: FindingTypeLintError, Severity: SeverityWarning},
	}

	result := parser.NewResult(findings)

	if result.BuildPassed {
		t.Error("expected BuildPassed = false")
	}
	if result.TestPassed {
		t.Error("expected TestPassed = false")
	}
	if result.LintPassed {
		t.Error("expected LintPassed = false")
	}

	if result.Summary == "All checks passed" {
		t.Error("summary should not be 'All checks passed'")
	}
}

func TestAtoiSafe(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"0", 0},
		{"42", 42},
		{"123", 123},
		{"abc", 0},
	}

	for _, tt := range tests {
		got := atoiSafe(tt.input)
		if got != tt.want {
			t.Errorf("atoiSafe(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
