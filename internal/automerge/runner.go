package automerge

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/internal/github"
)

// ProjectResolver maps project IDs to their working directories.
type ProjectResolver interface {
	GetProjectWorkDir(projectID string) string
	ListProjectIDs() []string
}

// PRClient abstracts GitHub PR operations for testability.
type PRClient interface {
	ListPRs(ctx context.Context, state string) ([]github.PullRequest, error)
	MergePR(ctx context.Context, number int, method string) error
}

// PRClientFactory creates a PRClient for a given working directory.
type PRClientFactory func(workDir string) PRClient

// defaultClientFactory returns a real github.Client.
func defaultClientFactory(workDir string) PRClient {
	return github.NewClient(workDir, "")
}

// Runner periodically sweeps open PRs across all projects and auto-merges
// those that meet readiness criteria: approved (or no review policy),
// mergeable, not draft, and from an agent branch prefix.
type Runner struct {
	projects      ProjectResolver
	clientFactory PRClientFactory
	mergeMethod   string
	stopCh        chan struct{}
}

// NewRunner creates an auto-merge runner with the default GitHub client factory.
func NewRunner(projects ProjectResolver) *Runner {
	return &Runner{
		projects:      projects,
		clientFactory: defaultClientFactory,
		mergeMethod:   "squash",
		stopCh:        make(chan struct{}),
	}
}

// Start runs the auto-merge sweep loop at the given interval until the
// context is cancelled or Stop is called.
func (r *Runner) Start(ctx context.Context, interval time.Duration) {
	log.Printf("[AutoMerge] Starting with %s interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.sweep(ctx)
		}
	}
}

// Stop signals the runner to exit its loop.
func (r *Runner) Stop() {
	close(r.stopCh)
}

func (r *Runner) sweep(ctx context.Context) {
	projectIDs := r.projects.ListProjectIDs()
	for _, pid := range projectIDs {
		workDir := r.projects.GetProjectWorkDir(pid)
		if workDir == "" {
			continue
		}
		merged, err := r.mergeReadyPRs(ctx, pid, workDir)
		if err != nil {
			log.Printf("[AutoMerge] Error checking PRs for project %s: %v", pid, err)
			continue
		}
		if merged > 0 {
			log.Printf("[AutoMerge] Merged %d PRs for project %s", merged, pid)
		}
	}
}

func (r *Runner) mergeReadyPRs(ctx context.Context, projectID, workDir string) (int, error) {
	client := r.clientFactory(workDir)

	prs, err := client.ListPRs(ctx, "open")
	if err != nil {
		return 0, err
	}

	merged := 0
	for _, pr := range prs {
		if !isAutoMergeable(pr) {
			continue
		}

		log.Printf("[AutoMerge] Merging PR #%d (%s) for project %s", pr.Number, pr.Title, projectID)
		if err := client.MergePR(ctx, pr.Number, r.mergeMethod); err != nil {
			log.Printf("[AutoMerge] Failed to merge PR #%d: %v", pr.Number, err)
			continue
		}
		merged++
	}
	return merged, nil
}

// isAutoMergeable returns true if a PR is ready for automatic merging.
// Criteria: not draft, mergeable, approved (or no review policy), agent branch.
func isAutoMergeable(pr github.PullRequest) bool {
	if pr.IsDraft {
		return false
	}

	if pr.Mergeable != "MERGEABLE" {
		return false
	}

	// APPROVED is the GitHub reviewDecision value for fully approved PRs.
	// Empty reviewDecision means no review policy is configured — treat as approved.
	if pr.ReviewDecision != "APPROVED" && pr.ReviewDecision != "" {
		return false
	}

	if !isAgentBranch(pr.HeadRef) {
		return false
	}

	if !ciPassed(pr.StatusChecks) {
		return false
	}
	return true
}

// isAgentBranch returns true if the branch name has an agent-created prefix.
func isAgentBranch(branch string) bool {
	prefixes := []string{"loom/", "agent/", "auto/", "fix/", "feat/"}
	for _, p := range prefixes {
		if strings.HasPrefix(branch, p) {
			return true
		}
	}
	return false
}

func ciPassed(checks []github.StatusCheck) bool {
	if len(checks) == 0 {
		return true
	}
	for _, c := range checks {
		if c.Status != "COMPLETED" {
			return false
		}
		if c.Conclusion != "SUCCESS" && c.Conclusion != "NEUTRAL" && c.Conclusion != "SKIPPED" {
			return false
		}
	}
	return true
}
