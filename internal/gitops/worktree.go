package gitops

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// GitWorktreeManager manages git worktrees for project and beads branches
type GitWorktreeManager struct {
	projectsRoot string // e.g., /app/data/projects
}

// NewGitWorktreeManager creates a new worktree manager
func NewGitWorktreeManager(projectsRoot string) *GitWorktreeManager {
	return &GitWorktreeManager{projectsRoot: projectsRoot}
}

// SetupBeadsWorktree creates isolated worktree for beads branch
// This allows concurrent access to main branch (for code) and beads branch (for bead metadata)
func (m *GitWorktreeManager) SetupBeadsWorktree(projectID, mainBranch, beadsBranch string) error {
	projectDir := filepath.Join(m.projectsRoot, projectID)
	mainWorktree := filepath.Join(projectDir, "main")
	beadsWorktree := filepath.Join(projectDir, "beads")

	// Ensure main worktree exists
	if _, err := os.Stat(mainWorktree); os.IsNotExist(err) {
		return fmt.Errorf("main worktree not found: %s", mainWorktree)
	}

	// Check if beads branch exists remotely
	checkCmd := exec.Command("git", "ls-remote", "--heads", "origin", beadsBranch)
	checkCmd.Dir = mainWorktree
	output, _ := checkCmd.CombinedOutput()

	branchExists := len(output) > 0

	if !branchExists {
		// Create orphan branch for beads
		if err := m.initializeBeadsBranch(mainWorktree, beadsBranch); err != nil {
			return fmt.Errorf("failed to initialize beads branch: %w", err)
		}
	}

	// Check if worktree already exists
	if _, err := os.Stat(beadsWorktree); !os.IsNotExist(err) {
		// Worktree exists, verify it's tracking the right branch
		return nil
	}

	// Create worktree for beads branch
	worktreeCmd := exec.Command("git", "worktree", "add", beadsWorktree, beadsBranch)
	worktreeCmd.Dir = mainWorktree
	if output, err := worktreeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %s - %w", output, err)
	}

	return nil
}

// initializeBeadsBranch creates orphan branch with initial structure
// Orphan branches have no parent commits and are used to store independent data
func (m *GitWorktreeManager) initializeBeadsBranch(repoPath, beadsBranch string) error {
	// Create orphan branch
	checkoutCmd := exec.Command("git", "checkout", "--orphan", beadsBranch)
	checkoutCmd.Dir = repoPath
	if err := checkoutCmd.Run(); err != nil {
		return err
	}

	// Remove all staged files
	resetCmd := exec.Command("git", "reset", "--hard")
	resetCmd.Dir = repoPath
	resetCmd.Run()

	// Create initial .beads structure
	beadsDir := filepath.Join(repoPath, ".beads", "beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create beads directory: %w", err)
	}

	// Create config file
	configPath := filepath.Join(repoPath, ".beads", "config.yaml")
	configContent := fmt.Sprintf(`# Beads configuration
version: 1
sync-branch: %s
`, beadsBranch)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create .gitignore to ensure .beads is tracked
	gitignorePath := filepath.Join(repoPath, ".beads", ".gitignore")
	gitignoreContent := `# Keep beads directory structure
!beads/
beads/.gitkeep
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// Initial commit
	addCmd := exec.Command("git", "add", ".beads")
	addCmd.Dir = repoPath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initialize beads branch")
	commitCmd.Dir = repoPath
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Try to push to remote (optional - may fail if no write access)
	pushCmd := exec.Command("git", "push", "-u", "origin", beadsBranch)
	pushCmd.Dir = repoPath
	if err := pushCmd.Run(); err != nil {
		// Log warning but don't fail - we can still use local branch
		log.Printf("Warning: Failed to push beads branch to remote: %v (branch will be local-only)", err)
	}

	// Return to main branch so worktree creation can use beads-sync
	checkoutMainCmd := exec.Command("git", "checkout", "main")
	checkoutMainCmd.Dir = repoPath
	if err := checkoutMainCmd.Run(); err != nil {
		return fmt.Errorf("failed to return to main branch: %w", err)
	}

	return nil
}

// GetWorktreePath returns path to specific worktree (main or beads)
func (m *GitWorktreeManager) GetWorktreePath(projectID, worktree string) string {
	return filepath.Join(m.projectsRoot, projectID, worktree)
}

// CleanupWorktree removes worktree (for cleanup/shutdown)
func (m *GitWorktreeManager) CleanupWorktree(projectID, worktree string) error {
	projectDir := filepath.Join(m.projectsRoot, projectID, "main")
	worktreePath := filepath.Join(m.projectsRoot, projectID, worktree)

	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s - %w", output, err)
	}

	return nil
}

// SyncBeadsBranch pulls latest beads from remote
// This ensures the local beads worktree is up-to-date with git remote
func (m *GitWorktreeManager) SyncBeadsBranch(projectID string) error {
	beadsWorktree := m.GetWorktreePath(projectID, "beads")

	pullCmd := exec.Command("git", "pull", "--rebase")
	pullCmd.Dir = beadsWorktree
	output, err := pullCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s - %w", output, err)
	}

	return nil
}
