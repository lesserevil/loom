package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

type mockBeadCreator struct {
	existingBeads []*models.Bead
	created       []*models.Bead
	createErr     error
	listErr       error
}

func (m *mockBeadCreator) CreateBead(title, description string, priority models.BeadPriority, beadType, projectID string) (*models.Bead, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	b := &models.Bead{
		ID:        "bd-new-" + title[:min(len(title), 10)],
		Title:     title,
		Status:    "open",
		Priority:  priority,
		ProjectID: projectID,
	}
	m.created = append(m.created, b)
	return b, nil
}

func (m *mockBeadCreator) GetBeadsByProject(projectID string) ([]*models.Bead, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.existingBeads, nil
}

func TestFileBeadsForFindings_NewFindings(t *testing.T) {
	mock := &mockBeadCreator{}
	activity := NewSelfAuditActivity(".")

	findings := []Finding{
		{Type: FindingTypeBuildError, Severity: SeverityError, Source: "go build", Message: "undefined: Foo"},
		{Type: FindingTypeTestFailure, Severity: SeverityError, Source: "go test", Message: "test failed", Rule: "TestBar"},
	}

	ids, err := activity.FileBeadsForFindings(context.Background(), mock, findings, "loom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 new beads, got %d", len(ids))
	}
	if len(mock.created) != 2 {
		t.Errorf("expected 2 created beads, got %d", len(mock.created))
	}
}

func TestFileBeadsForFindings_Deduplication(t *testing.T) {
	activity := NewSelfAuditActivity(".")
	finding := Finding{
		Type: FindingTypeBuildError, Severity: SeverityError,
		Source: "go build", Message: "undefined: Foo",
	}
	title := activity.findingToTitle(finding)

	mock := &mockBeadCreator{
		existingBeads: []*models.Bead{
			{ID: "bd-existing", Title: title, Status: "open"},
		},
	}

	ids, err := activity.FileBeadsForFindings(context.Background(), mock, []Finding{finding}, "loom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 new beads (duplicate), got %d", len(ids))
	}
	if len(mock.created) != 0 {
		t.Errorf("expected 0 created beads, got %d", len(mock.created))
	}
}

func TestFileBeadsForFindings_SkipsClosedDuplicates(t *testing.T) {
	activity := NewSelfAuditActivity(".")
	finding := Finding{
		Type: FindingTypeBuildError, Severity: SeverityError,
		Source: "go build", Message: "undefined: Foo",
	}
	title := activity.findingToTitle(finding)

	mock := &mockBeadCreator{
		existingBeads: []*models.Bead{
			{ID: "bd-closed", Title: title, Status: "closed"},
		},
	}

	ids, err := activity.FileBeadsForFindings(context.Background(), mock, []Finding{finding}, "loom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 new bead (closed dup re-filed), got %d", len(ids))
	}
}

func TestFileBeadsForFindings_ListError(t *testing.T) {
	mock := &mockBeadCreator{listErr: errors.New("db down")}
	activity := NewSelfAuditActivity(".")

	_, err := activity.FileBeadsForFindings(context.Background(), mock, []Finding{{Message: "x"}}, "loom")
	if err == nil {
		t.Error("expected error when GetBeadsByProject fails")
	}
}

func TestFileBeadsForFindings_CreateErrorContinues(t *testing.T) {
	mock := &mockBeadCreator{createErr: errors.New("write failed")}
	activity := NewSelfAuditActivity(".")

	findings := []Finding{
		{Type: FindingTypeBuildError, Severity: SeverityError, Source: "go build", Message: "err1"},
		{Type: FindingTypeBuildError, Severity: SeverityError, Source: "go build", Message: "err2"},
	}

	ids, err := activity.FileBeadsForFindings(context.Background(), mock, findings, "loom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 beads filed on create error, got %d", len(ids))
	}
}

func TestFileBeadsForFindings_WarningPriority(t *testing.T) {
	mock := &mockBeadCreator{}
	activity := NewSelfAuditActivity(".")

	findings := []Finding{
		{Type: FindingTypeLintError, Severity: SeverityWarning, Source: "golangci-lint", Message: "exported func"},
	}

	_, err := activity.FileBeadsForFindings(context.Background(), mock, findings, "loom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.created) != 1 {
		t.Fatalf("expected 1 created bead, got %d", len(mock.created))
	}
	if mock.created[0].Priority != models.BeadPriorityP2 {
		t.Errorf("expected P2 priority for warning, got %v", mock.created[0].Priority)
	}
}

func TestFindingToTitle_Types(t *testing.T) {
	activity := NewSelfAuditActivity(".")

	tests := []struct {
		finding    Finding
		wantPrefix string
	}{
		{Finding{Type: FindingTypeBuildError, Message: "bad"}, "[auto-audit] Build error:"},
		{Finding{Type: FindingTypeTestFailure, Message: "bad"}, "[auto-audit] Test failure:"},
		{Finding{Type: FindingTypeLintError, Message: "bad"}, "[auto-audit] Lint warning:"},
		{Finding{Type: FindingTypeLintError, Message: "bad", Rule: "errcheck"}, "[auto-audit] Lint: errcheck"},
	}

	for _, tt := range tests {
		title := activity.findingToTitle(tt.finding)
		if len(title) == 0 {
			t.Error("expected non-empty title")
		}
		if !contains(title, tt.wantPrefix) {
			t.Errorf("title %q should contain %q", title, tt.wantPrefix)
		}
	}
}

func TestTruncateTitle(t *testing.T) {
	short := "short"
	if truncateTitle(short) != short {
		t.Errorf("expected %q, got %q", short, truncateTitle(short))
	}

	long := "this is a very long message that goes well beyond the eighty character limit for bead titles because we want truncation"
	truncated := truncateTitle(long)
	if len(truncated) > 80 {
		t.Errorf("expected truncated to <=80 chars, got %d", len(truncated))
	}
	if truncated[len(truncated)-3:] != "..." {
		t.Error("expected truncated string to end with ...")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstr(s, substr))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
