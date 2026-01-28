package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// Manager handles git operations for managed projects
type Manager struct {
	baseWorkDir   string // Base directory for all project clones (e.g., /app/src)
	projectKeyDir string // Base directory for per-project SSH keys
}

// NewManager creates a new git operations manager
func NewManager(baseWorkDir, projectKeyDir string) (*Manager, error) {
	// Ensure base work directory exists
	if err := os.MkdirAll(baseWorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base work directory: %w", err)
	}

	if projectKeyDir == "" {
		projectKeyDir = filepath.Join("/app/data", "projects")
	}
	if err := os.MkdirAll(projectKeyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project key directory: %w", err)
	}

	return &Manager{
		baseWorkDir:   baseWorkDir,
		projectKeyDir: projectKeyDir,
	}, nil
}

// CloneProject clones a project's git repository into its work directory
func (m *Manager) CloneProject(ctx context.Context, project *models.Project) error {
	if project.GitRepo == "" {
		return fmt.Errorf("project %s has no git_repo configured", project.ID)
	}

	workDir := m.GetProjectWorkDir(project.ID)

	// Check if already cloned
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		return fmt.Errorf("project %s already cloned at %s", project.ID, workDir)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		return fmt.Errorf("failed to create work directory parent: %w", err)
	}

	// Build clone command
	args := []string{"clone"}

	// Add branch if specified
	if project.Branch != "" {
		args = append(args, "--branch", project.Branch)
	}

	// Single branch to save space
	args = append(args, "--single-branch", project.GitRepo, workDir)

	// Execute git clone
	cmd := exec.CommandContext(ctx, "git", args...)

	// Configure auth if needed
	if err := m.configureAuth(cmd, project); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	// Update project metadata
	project.WorkDir = workDir
	project.LastSyncAt = timePtr(time.Now())

	// Get initial commit hash
	if hash, err := m.GetCurrentCommit(workDir); err == nil {
		project.LastCommitHash = hash
	}

	return nil
}

// PullProject pulls latest changes from remote
func (m *Manager) PullProject(ctx context.Context, project *models.Project) error {
	workDir := m.GetProjectWorkDir(project.ID)

	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("project %s not cloned, call CloneProject first", project.ID)
	}

	cmd := exec.CommandContext(ctx, "git", "pull", "--rebase")
	cmd.Dir = workDir

	if err := m.configureAuth(cmd, project); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w\nOutput: %s", err, string(output))
	}

	// Update metadata
	project.LastSyncAt = timePtr(time.Now())
	if hash, err := m.GetCurrentCommit(workDir); err == nil {
		project.LastCommitHash = hash
	}

	return nil
}

// CommitChanges commits all changes in the project work directory
func (m *Manager) CommitChanges(ctx context.Context, project *models.Project, message, authorName, authorEmail string) error {
	workDir := m.GetProjectWorkDir(project.ID)

	// Stage all changes
	if err := m.runGitCommand(ctx, workDir, "add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are changes to commit
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = workDir
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return nil // No changes to commit
	}

	// Commit with author info
	args := []string{"commit", "-m", message}
	if authorName != "" && authorEmail != "" {
		args = append(args, "--author", fmt.Sprintf("%s <%s>", authorName, authorEmail))
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir
	if authorName != "" && authorEmail != "" {
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_AUTHOR_NAME=%s", authorName),
			fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", authorEmail),
			fmt.Sprintf("GIT_COMMITTER_NAME=%s", authorName),
			fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", authorEmail),
		)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	// Update commit hash
	if hash, err := m.GetCurrentCommit(workDir); err == nil {
		project.LastCommitHash = hash
	}

	return nil
}

// PushChanges pushes committed changes to remote
func (m *Manager) PushChanges(ctx context.Context, project *models.Project) error {
	workDir := m.GetProjectWorkDir(project.ID)

	cmd := exec.CommandContext(ctx, "git", "push")
	cmd.Dir = workDir

	if err := m.configureAuth(cmd, project); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Status returns git status for a project workdir.
func (m *Manager) Status(ctx context.Context, projectID string) (string, error) {
	workDir := m.GetProjectWorkDir(projectID)
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("project %s not cloned", projectID)
	}
	output, err := m.runGitCommandWithOutput(ctx, workDir, "status", "-sb")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// Diff returns git diff for a project workdir.
func (m *Manager) Diff(ctx context.Context, projectID string) (string, error) {
	workDir := m.GetProjectWorkDir(projectID)
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("project %s not cloned", projectID)
	}
	output, err := m.runGitCommandWithOutput(ctx, workDir, "diff")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// GetCurrentCommit returns the current commit SHA
func (m *Manager) GetCurrentCommit(workDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetProjectWorkDir returns the work directory path for a project
func (m *Manager) GetProjectWorkDir(projectID string) string {
	// Always use baseWorkDir/projectID for cloned projects
	// The special case for agenticorp-self was removed because in Docker,
	// the repo is cloned separately to baseWorkDir/agenticorp-self even though
	// baseWorkDir/.git may exist from the image build.
	return filepath.Join(m.baseWorkDir, projectID)
}

// LoadBeadsFromProject loads beads from a project's cloned repository
func (m *Manager) LoadBeadsFromProject(project *models.Project) ([]models.Bead, error) {
	workDir := m.GetProjectWorkDir(project.ID)
	beadsDir := filepath.Join(workDir, project.BeadsPath, "beads")

	// Check if beads directory exists
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		return nil, nil // No beads directory, return empty
	}

	// This would integrate with the existing bead loading logic
	// For now, return placeholder - actual implementation would use
	// the existing LoadBeadsFromFilesystem function
	return nil, nil
}

// configureAuth configures git authentication for a command
func (m *Manager) configureAuth(cmd *exec.Cmd, project *models.Project) error {
	switch project.GitAuthMethod {
	case models.GitAuthNone:
		// No auth needed
		return nil

	case models.GitAuthSSH:
		publicKey, err := m.EnsureProjectSSHKey(project.ID)
		if err != nil {
			return err
		}
		_ = publicKey
		sshKeyPath := m.projectPrivateKeyPath(project.ID)
		if _, err := os.Stat(sshKeyPath); err != nil {
			return fmt.Errorf("ssh key not found for project %s: %w", project.ID, err)
		}
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env,
			"GIT_TERMINAL_PROMPT=0",
			fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes -o UserKnownHostsFile=/home/agenticorp/.ssh/known_hosts", sshKeyPath),
		)
		return nil

	case models.GitAuthToken:
		// For HTTPS with token, we could inject into URL or use credential helper
		// This is a simplified approach - production would use credential helper
		return nil

	case models.GitAuthBasic:
		// Would integrate with secrets store for username/password
		return nil

	default:
		return fmt.Errorf("unsupported auth method: %s", project.GitAuthMethod)
	}
}

// runGitCommand is a helper to run git commands in a work directory
func (m *Manager) runGitCommand(ctx context.Context, workDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (m *Manager) runGitCommandWithOutput(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nOutput: %s", strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

func (m *Manager) projectKeyDirForProject(projectID string) string {
	return filepath.Join(m.projectKeyDir, projectID, "ssh")
}

func (m *Manager) projectPrivateKeyPath(projectID string) string {
	return filepath.Join(m.projectKeyDirForProject(projectID), "id_ed25519")
}

func (m *Manager) projectPublicKeyPath(projectID string) string {
	return m.projectPrivateKeyPath(projectID) + ".pub"
}

// EnsureProjectSSHKey ensures an SSH keypair exists for the project and returns the public key.
func (m *Manager) EnsureProjectSSHKey(projectID string) (string, error) {
	if projectID == "" {
		return "", fmt.Errorf("project ID is required")
	}

	keyDir := m.projectKeyDirForProject(projectID)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create project ssh directory: %w", err)
	}

	privatePath := m.projectPrivateKeyPath(projectID)
	publicPath := m.projectPublicKeyPath(projectID)
	if _, err := os.Stat(privatePath); os.IsNotExist(err) {
		if err := m.generateSSHKeyPair(privatePath); err != nil {
			return "", err
		}
	}

	if _, err := os.Stat(publicPath); os.IsNotExist(err) {
		if err := m.writePublicKeyFromPrivate(privatePath, publicPath); err != nil {
			return "", err
		}
	}

	keyBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	return strings.TrimSpace(string(keyBytes)), nil
}

// GetProjectPublicKey returns the project's public SSH key, creating it if needed.
func (m *Manager) GetProjectPublicKey(projectID string) (string, error) {
	return m.EnsureProjectSSHKey(projectID)
}

// RotateProjectSSHKey regenerates the project's SSH keypair and returns the new public key.
func (m *Manager) RotateProjectSSHKey(projectID string) (string, error) {
	if projectID == "" {
		return "", fmt.Errorf("project ID is required")
	}
	privatePath := m.projectPrivateKeyPath(projectID)
	publicPath := m.projectPublicKeyPath(projectID)
	_ = os.Remove(privatePath)
	_ = os.Remove(publicPath)
	if err := m.generateSSHKeyPair(privatePath); err != nil {
		return "", err
	}
	keyBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return strings.TrimSpace(string(keyBytes)), nil
}

func (m *Manager) generateSSHKeyPair(privatePath string) error {
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", privatePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate ssh key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Chmod(privatePath, 0600); err != nil {
		return fmt.Errorf("failed to set ssh key permissions: %w", err)
	}
	return nil
}

func (m *Manager) writePublicKeyFromPrivate(privatePath, publicPath string) error {
	cmd := exec.Command("ssh-keygen", "-y", "-f", privatePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to derive public key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.WriteFile(publicPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	return nil
}

// CheckRemoteAccess verifies that the configured git auth can access the remote.
func (m *Manager) CheckRemoteAccess(ctx context.Context, project *models.Project) error {
	if project == nil {
		return fmt.Errorf("project is required")
	}
	if project.GitRepo == "" || project.GitRepo == "." {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "ls-remote", project.GitRepo, "HEAD")
	if err := m.configureAuth(cmd, project); err != nil {
		return err
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git ls-remote failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Helper to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
