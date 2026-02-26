package github

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	ID         int64
	Name       string
	Status     string
	Conclusion string
	URL        string
}

// RepoInfo represents basic information about a GitHub repository.
type RepoInfo struct {
	NameWithOwner string
	DefaultBranch string
	Description   string
	URL           string
	IsPrivate     bool
}

// Issue represents a GitHub issue.
type Issue struct {
	Number int
	Title  string
	Body   string
	State  string
	URL    string
	Author string
	Labels []string
}
