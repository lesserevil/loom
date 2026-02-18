// Package paths provides centralized path management for Loom
// This ensures consistent path handling across the codebase
package paths

import (
	"fmt"
	"path/filepath"
)

// PathManager provides centralized path management
type PathManager struct {
	dataRoot     string // Root data directory (e.g., /app/data)
	projectsRoot string // Projects directory (e.g., /app/data/projects)
	keysRoot     string // SSH keys directory (e.g., /app/data/keys)
}

// NewPathManager creates a new path manager with the given root directory
func NewPathManager(dataRoot string) *PathManager {
	return &PathManager{
		dataRoot:     dataRoot,
		projectsRoot: filepath.Join(dataRoot, "projects"),
		keysRoot:     filepath.Join(dataRoot, "keys"),
	}
}

// DataRoot returns the root data directory
func (pm *PathManager) DataRoot() string {
	return pm.dataRoot
}

// ProjectsRoot returns the projects root directory
func (pm *PathManager) ProjectsRoot() string {
	return pm.projectsRoot
}

// KeysRoot returns the SSH keys root directory
func (pm *PathManager) KeysRoot() string {
	return pm.keysRoot
}

// ProjectDir returns the project directory path
func (pm *PathManager) ProjectDir(projectID string) string {
	return filepath.Join(pm.projectsRoot, projectID)
}

// ProjectMainWorktree returns the main worktree path for a project
func (pm *PathManager) ProjectMainWorktree(projectID string) string {
	return filepath.Join(pm.projectsRoot, projectID, "main")
}

// ProjectBeadsWorktree returns the beads worktree path for a project
func (pm *PathManager) ProjectBeadsWorktree(projectID string) string {
	return filepath.Join(pm.projectsRoot, projectID, "beads")
}

// ProjectBeadsPath returns the beads directory path within the beads worktree
func (pm *PathManager) ProjectBeadsPath(projectID, beadsPath string) string {
	return filepath.Join(pm.ProjectBeadsWorktree(projectID), beadsPath)
}

// ProjectSSHKeyDir returns the SSH key directory for a project
func (pm *PathManager) ProjectSSHKeyDir(projectID string) string {
	return filepath.Join(pm.keysRoot, projectID, "ssh")
}

// ProjectSSHPrivateKey returns the private SSH key path for a project
func (pm *PathManager) ProjectSSHPrivateKey(projectID string) string {
	return filepath.Join(pm.ProjectSSHKeyDir(projectID), "id_ed25519")
}

// ProjectSSHPublicKey returns the public SSH key path for a project
func (pm *PathManager) ProjectSSHPublicKey(projectID string) string {
	return filepath.Join(pm.ProjectSSHPrivateKey(projectID) + ".pub")
}

// ProjectContainerCompose returns the docker-compose file path for a project container
func (pm *PathManager) ProjectContainerCompose(projectID string) string {
	return filepath.Join(pm.ProjectDir(projectID), "docker-compose.yml")
}

// ProjectContainerWorkspace returns the workspace volume name for a project container
func (pm *PathManager) ProjectContainerWorkspace(projectID string) string {
	return fmt.Sprintf("loom-project-%s-workspace", projectID)
}

// ProjectContainerName returns the container name for a project
func (pm *PathManager) ProjectContainerName(projectID string) string {
	return fmt.Sprintf("loom-project-%s", projectID)
}

// ProjectContainerImageName returns the Docker image name for a project
func (pm *PathManager) ProjectContainerImageName(projectID string) string {
	return fmt.Sprintf("loom-project:%s", projectID)
}

// Default path manager instance using standard Loom paths
var Default = NewPathManager("/app/data")
