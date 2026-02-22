package automerge

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/github"
)

// --- mocks ---

type mockProjectResolver struct {
	projects map[string]string // projectID -> workDir
}

func (m *mockProjectResolver) GetProjectWorkDir(projectID string) string {
	return m.projects[projectID]
}

func (m *mockProjectResolver) ListProjectIDs() []string {
	ids := make([]string, 0, len(m.projects))
	for id := range m.projects {
		ids = append(ids, id)
	}
	return ids
}

type mockPRClient struct {
	mu       sync.Mutex
	prs      []github.PullRequest
	listErr  error
	merged   []int
	mergeErr map[int]error
}

func (m *mockPRClient) ListPRs(_ context.Context, _ string) ([]github.PullRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.prs, nil
}

func (m *mockPRClient) MergePR(_ context.Context, number int, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mergeErr != nil {
		if err, ok := m.mergeErr[number]; ok {
			return err
		}
	}
	m.merged = append(m.merged, number)
	return nil
}

func (m *mockPRClient) getMerged() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]int, len(m.merged))
	copy(cp, m.merged)
	return cp
}

// --- isAgentBranch tests ---

func TestIsAgentBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   bool
	}{
		{"loom/fix-build", true},
		{"agent/coder-123", true},
		{"auto/lint-fix", true},
		{"fix/nil-pointer", true},
		{"feat/new-api", true},
		{"main", false},
		{"develop", false},
		{"feature/user-auth", false},
		{"hotfix/urgent", false},
		{"", false},
		{"loom", false},
		{"agent", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			if got := isAgentBranch(tt.branch); got != tt.want {
				t.Errorf("isAgentBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

// --- isAutoMergeable tests ---

func TestIsAutoMergeable(t *testing.T) {
	readyPR := github.PullRequest{
		Number:         1,
		HeadRef:        "loom/fix-build",
		Mergeable:      "MERGEABLE",
		ReviewDecision: "APPROVED",
		IsDraft:        false,
	}

	tests := []struct {
		name   string
		modify func(github.PullRequest) github.PullRequest
		want   bool
	}{
		{"fully ready PR", func(pr github.PullRequest) github.PullRequest { return pr }, true},
		{"no review policy (empty reviewDecision)", func(pr github.PullRequest) github.PullRequest {
			pr.ReviewDecision = ""
			return pr
		}, true},
		{"draft PR rejected", func(pr github.PullRequest) github.PullRequest {
			pr.IsDraft = true
			return pr
		}, false},
		{"not mergeable", func(pr github.PullRequest) github.PullRequest {
			pr.Mergeable = "CONFLICTING"
			return pr
		}, false},
		{"changes requested", func(pr github.PullRequest) github.PullRequest {
			pr.ReviewDecision = "CHANGES_REQUESTED"
			return pr
		}, false},
		{"review required", func(pr github.PullRequest) github.PullRequest {
			pr.ReviewDecision = "REVIEW_REQUIRED"
			return pr
		}, false},
		{"non-agent branch", func(pr github.PullRequest) github.PullRequest {
			pr.HeadRef = "feature/manual-work"
			return pr
		}, false},
		{"human main branch", func(pr github.PullRequest) github.PullRequest {
			pr.HeadRef = "main"
			return pr
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := tt.modify(readyPR)
			if got := isAutoMergeable(pr); got != tt.want {
				t.Errorf("isAutoMergeable() = %v, want %v (pr: %+v)", got, tt.want, pr)
			}
		})
	}
}

func TestCIPassed(t *testing.T) {
	tests := []struct {
		name   string
		checks []github.StatusCheck
		want   bool
	}{
		{"no checks (no CI configured)", nil, true},
		{"empty checks", []github.StatusCheck{}, true},
		{"all success", []github.StatusCheck{
			{Name: "build", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		}, true},
		{"neutral is ok", []github.StatusCheck{
			{Name: "lint", Status: "COMPLETED", Conclusion: "NEUTRAL"},
		}, true},
		{"skipped is ok", []github.StatusCheck{
			{Name: "deploy", Status: "COMPLETED", Conclusion: "SKIPPED"},
		}, true},
		{"mixed passing", []github.StatusCheck{
			{Name: "build", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "COMPLETED", Conclusion: "NEUTRAL"},
			{Name: "optional", Status: "COMPLETED", Conclusion: "SKIPPED"},
		}, true},
		{"one failure", []github.StatusCheck{
			{Name: "build", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "test", Status: "COMPLETED", Conclusion: "FAILURE"},
		}, false},
		{"in progress", []github.StatusCheck{
			{Name: "build", Status: "IN_PROGRESS", Conclusion: ""},
		}, false},
		{"queued", []github.StatusCheck{
			{Name: "build", Status: "QUEUED", Conclusion: ""},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ciPassed(tt.checks); got != tt.want {
				t.Errorf("ciPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Runner lifecycle tests ---

func TestNewRunner(t *testing.T) {
	projects := &mockProjectResolver{projects: map[string]string{"p1": "/tmp/p1"}}
	r := NewRunner(projects)

	if r.projects != projects {
		t.Error("projects not set")
	}
	if r.mergeMethod != "squash" {
		t.Errorf("mergeMethod = %q, want squash", r.mergeMethod)
	}
	if r.clientFactory == nil {
		t.Error("clientFactory is nil")
	}
	if r.stopCh == nil {
		t.Error("stopCh is nil")
	}
}

func TestRunner_Stop(t *testing.T) {
	r := NewRunner(&mockProjectResolver{projects: map[string]string{}})
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		r.Start(ctx, 1*time.Hour)
		close(done)
	}()

	r.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop within 2s")
	}
}

func TestRunner_ContextCancel(t *testing.T) {
	r := NewRunner(&mockProjectResolver{projects: map[string]string{}})
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		r.Start(ctx, 1*time.Hour)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop on context cancel within 2s")
	}
}

// --- sweep / mergeReadyPRs tests ---

func TestSweep_MergesReadyPRs(t *testing.T) {
	mock := &mockPRClient{
		prs: []github.PullRequest{
			{Number: 10, HeadRef: "loom/fix-a", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
			{Number: 20, HeadRef: "loom/fix-b", Mergeable: "MERGEABLE", ReviewDecision: ""},
		},
	}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": "/tmp/proj1"}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged PRs, got %d: %v", len(merged), merged)
	}
}

func TestSweep_SkipsNonMergeablePRs(t *testing.T) {
	mock := &mockPRClient{
		prs: []github.PullRequest{
			{Number: 1, HeadRef: "loom/fix-a", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
			{Number: 2, HeadRef: "loom/fix-b", Mergeable: "CONFLICTING", ReviewDecision: "APPROVED"},
			{Number: 3, HeadRef: "feature/manual", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
			{Number: 4, HeadRef: "loom/fix-c", Mergeable: "MERGEABLE", ReviewDecision: "CHANGES_REQUESTED"},
			{Number: 5, HeadRef: "loom/fix-d", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED", IsDraft: true},
		},
	}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": "/tmp/proj1"}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 1 || merged[0] != 1 {
		t.Fatalf("expected only PR #1 merged, got %v", merged)
	}
}

func TestSweep_SkipsEmptyWorkDir(t *testing.T) {
	mock := &mockPRClient{
		prs: []github.PullRequest{
			{Number: 1, HeadRef: "loom/fix", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
		},
	}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": ""}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 0 {
		t.Fatalf("expected no merges for empty workDir, got %v", merged)
	}
}

func TestSweep_MultipleProjects(t *testing.T) {
	clients := map[string]*mockPRClient{
		"/tmp/p1": {prs: []github.PullRequest{
			{Number: 1, HeadRef: "fix/a", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
		}},
		"/tmp/p2": {prs: []github.PullRequest{
			{Number: 2, HeadRef: "agent/b", Mergeable: "MERGEABLE", ReviewDecision: ""},
		}},
	}

	projects := &mockProjectResolver{projects: map[string]string{
		"p1": "/tmp/p1",
		"p2": "/tmp/p2",
	}}
	r := NewRunner(projects)
	r.clientFactory = func(workDir string) PRClient {
		return clients[workDir]
	}

	r.sweep(context.Background())

	total := 0
	for _, c := range clients {
		total += len(c.getMerged())
	}
	if total != 2 {
		t.Fatalf("expected 2 total merges across projects, got %d", total)
	}
}

func TestSweep_ListPRsError(t *testing.T) {
	mock := &mockPRClient{listErr: fmt.Errorf("gh: network error")}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": "/tmp/proj1"}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	// Should not panic; error is logged.
	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 0 {
		t.Fatalf("expected no merges on ListPRs error, got %v", merged)
	}
}

func TestSweep_MergePRError_ContinuesOthers(t *testing.T) {
	mock := &mockPRClient{
		prs: []github.PullRequest{
			{Number: 1, HeadRef: "loom/a", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
			{Number: 2, HeadRef: "loom/b", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
			{Number: 3, HeadRef: "loom/c", Mergeable: "MERGEABLE", ReviewDecision: "APPROVED"},
		},
		mergeErr: map[int]error{2: fmt.Errorf("merge conflict")},
	}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": "/tmp/proj1"}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged (skipping #2), got %v", merged)
	}
	for _, n := range merged {
		if n == 2 {
			t.Fatal("PR #2 should not have been merged")
		}
	}
}

func TestSweep_NoPRs(t *testing.T) {
	mock := &mockPRClient{prs: []github.PullRequest{}}

	projects := &mockProjectResolver{projects: map[string]string{"proj1": "/tmp/proj1"}}
	r := NewRunner(projects)
	r.clientFactory = func(_ string) PRClient { return mock }

	r.sweep(context.Background())

	merged := mock.getMerged()
	if len(merged) != 0 {
		t.Fatalf("expected no merges for empty PR list, got %v", merged)
	}
}

func TestSweep_NoProjects(t *testing.T) {
	projects := &mockProjectResolver{projects: map[string]string{}}
	r := NewRunner(projects)

	// Should not panic with no projects.
	r.sweep(context.Background())
}
