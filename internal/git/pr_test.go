package git

import (
	"testing"
)

func TestCreatePRRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePRRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with all fields",
			req: CreatePRRequest{
				BeadID:    "bead-123",
				Title:     "Add feature X",
				Body:      "This PR adds feature X",
				Base:      "main",
				Branch:    "agent/bead-123/feature-x",
				Reviewers: []string{"reviewer1"},
				Draft:     false,
			},
			wantErr: false,
		},
		{
			name: "valid request with minimal fields",
			req: CreatePRRequest{
				BeadID: "bead-456",
				Branch: "agent/bead-456/fix-bug",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - just checking struct can be created
			if tt.req.BeadID == "" {
				t.Error("BeadID should not be empty")
			}
		})
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int
	}{
		{
			name: "github pr url",
			url:  "https://github.com/owner/repo/pull/123",
			want: 123,
		},
		{
			name: "github pr url with trailing slash",
			url:  "https://github.com/owner/repo/pull/456/",
			want: 456,
		},
		{
			name: "invalid url",
			url:  "https://github.com/owner/repo",
			want: 0,
		},
		{
			name: "not a pr url",
			url:  "https://github.com/owner/repo/issues/789",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPRNumber(tt.url)
			if got != tt.want {
				t.Errorf("extractPRNumber(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

func TestCreatePR_BranchValidation(t *testing.T) {
	// This test requires a mock service - for now just test branch validation logic
	testCases := []struct {
		branch  string
		isValid bool
	}{
		{"agent/bead-123/feature", true},
		{"main", false},
		{"feature/something", false},
		{"agent/fix", true},
	}

	for _, tc := range testCases {
		t.Run(tc.branch, func(t *testing.T) {
			// Validate branch name starts with agent/
			hasPrefix := len(tc.branch) >= 6 && tc.branch[0:6] == "agent/"
			if hasPrefix != tc.isValid {
				t.Errorf("Branch %q validation = %v, want %v", tc.branch, hasPrefix, tc.isValid)
			}
		})
	}
}

// Note: Full integration tests for CreatePR require:
// 1. Test git repository with gh CLI configured
// 2. GitHub authentication
// 3. Remote repository with permissions
// These should be run separately with -tags=integration
func TestCreatePR_Integration(t *testing.T) {
	t.Skip("Integration test - requires gh CLI and GitHub auth")

	// Example integration test structure:
	// 1. Create test repo
	// 2. Create agent branch
	// 3. Make changes
	// 4. Commit
	// 5. Create PR
	// 6. Verify PR exists
	// 7. Clean up
}

func TestIsGhCLIAvailable(t *testing.T) {
	// This test checks if gh CLI is available
	// It will pass if gh is installed and authenticated, skip otherwise
	available := isGhCLIAvailable()
	t.Logf("gh CLI available: %v", available)

	// Don't fail the test if gh is not available - it's environment dependent
	// Just log the result for informational purposes
}
