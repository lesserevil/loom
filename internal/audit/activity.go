package audit

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

type SelfAuditActivity struct {
	projectPath string
	parser      *Parser
}

func NewSelfAuditActivity(projectPath string) *SelfAuditActivity {
	return &SelfAuditActivity{
		projectPath: projectPath,
		parser:      NewParser(),
	}
}

type SelfAuditInput struct {
	ProjectID   string
	ProjectPath string
}

type SelfAuditOutput struct {
	Result       *Result
	NewBeads     []string
	ExistingOpen int
}

func (a *SelfAuditActivity) RunSelfAudit(ctx context.Context, input SelfAuditInput) (*SelfAuditOutput, error) {
	projectPath := input.ProjectPath
	if projectPath == "" {
		projectPath = a.projectPath
	}
	if projectPath == "" {
		projectPath = "."
	}

	var allFindings []Finding

	buildOut, buildErr := a.runCommand(ctx, "go", []string{"build", "./..."}, projectPath, 5*time.Minute)
	if buildErr != nil {
		allFindings = append(allFindings, a.parser.ParseGoBuild(buildOut)...)
	}

	testOut, testErr := a.runCommand(ctx, "go", []string{"test", "-short", "./..."}, projectPath, 10*time.Minute)
	if testErr != nil {
		allFindings = append(allFindings, a.parser.ParseGoTest(testOut)...)
	}

	lintOut, lintErr := a.runCommand(ctx, "golangci-lint", []string{"run", "--timeout=5m"}, projectPath, 10*time.Minute)
	if lintErr != nil && strings.TrimSpace(lintOut) != "" {
		allFindings = append(allFindings, a.parser.ParseGoLint(lintOut)...)
	}

	result := a.parser.NewResult(allFindings)

	return &SelfAuditOutput{
		Result:       result,
		NewBeads:     []string{},
		ExistingOpen: 0,
	}, nil
}

func (a *SelfAuditActivity) runCommand(ctx context.Context, name string, args []string, dir string, timeout time.Duration) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CI=true")

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		return output, err
	}
	return output, nil
}

type BeadCreator interface {
	CreateBead(title, description string, priority models.BeadPriority, beadType, projectID string) (*models.Bead, error)
	GetBeadsByProject(projectID string) ([]*models.Bead, error)
}

func (a *SelfAuditActivity) FileBeadsForFindings(ctx context.Context, beadCreator BeadCreator, findings []Finding, projectID string) ([]string, error) {
	existingBeads, err := beadCreator.GetBeadsByProject(projectID)
	if err != nil {
		return nil, err
	}

	existingTitles := make(map[string]bool)
	for _, b := range existingBeads {
		if b.Status == "open" || b.Status == "in_progress" {
			existingTitles[b.Title] = true
		}
	}

	var newBeadIDs []string

	for _, f := range findings {
		title := a.findingToTitle(f)
		if existingTitles[title] {
			continue
		}

		desc := a.findingToDescription(f)
		priority := models.BeadPriorityP1
		if f.Severity == SeverityWarning {
			priority = models.BeadPriorityP2
		}

		bead, err := beadCreator.CreateBead(title, desc, priority, "bug", projectID)
		if err != nil {
			continue
		}

		newBeadIDs = append(newBeadIDs, bead.ID)
		existingTitles[title] = true
	}

	return newBeadIDs, nil
}

func (a *SelfAuditActivity) findingToTitle(f Finding) string {
	prefix := "[auto-audit]"
	switch f.Type {
	case FindingTypeBuildError:
		return prefix + " Build error: " + truncateTitle(f.Message)
	case FindingTypeTestFailure:
		return prefix + " Test failure: " + truncateTitle(f.Message)
	case FindingTypeLintError:
		if f.Rule != "" {
			return prefix + " Lint: " + f.Rule + " - " + truncateTitle(f.Message)
		}
		return prefix + " Lint warning: " + truncateTitle(f.Message)
	default:
		return prefix + " " + truncateTitle(f.Message)
	}
}

func (a *SelfAuditActivity) findingToDescription(f Finding) string {
	var b strings.Builder
	b.WriteString("Auto-filed from self-audit\n\n")
	b.WriteString("Source: " + f.Source + "\n")
	b.WriteString("Type: " + string(f.Type) + "\n")
	b.WriteString("Severity: " + string(f.Severity) + "\n")
	if f.File != "" {
		b.WriteString("File: " + f.File + "\n")
	}
	if f.Line > 0 {
		b.WriteString("Line: " + strconv.Itoa(f.Line) + "\n")
	}
	b.WriteString("\n---\n\n")
	b.WriteString("Message: " + f.Message + "\n")
	if f.Rule != "" {
		b.WriteString("\nRule: " + f.Rule + "\n")
	}
	return b.String()
}

func truncateTitle(s string) string {
	if len(s) <= 80 {
		return s
	}
	return s[:77] + "..."
}
