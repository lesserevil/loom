package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BootstrapRequest contains the parameters for bootstrapping a new project
type BootstrapRequest struct {
	GitHubURL string `json:"github_url"`
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	PRDText   string `json:"prd_text,omitempty"`   // PRD as text
	PRDFile   []byte `json:"prd_file,omitempty"`   // Or uploaded file content
}

// BootstrapResult contains the result of a bootstrap operation
type BootstrapResult struct {
	ProjectID   string `json:"project_id"`
	Status      string `json:"status"`
	InitialBead string `json:"initial_bead_id,omitempty"` // PM's PRD expansion bead
	Error       string `json:"error,omitempty"`
}

// BootstrapService handles project bootstrap operations
type BootstrapService struct {
	projectManager *Manager
	templateDir    string
	workspaceDir   string
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService(pm *Manager, templateDir, workspaceDir string) *BootstrapService {
	return &BootstrapService{
		projectManager: pm,
		templateDir:    templateDir,
		workspaceDir:   workspaceDir,
	}
}

// Bootstrap creates a new project from a PRD
func (bs *BootstrapService) Bootstrap(ctx context.Context, req BootstrapRequest) (*BootstrapResult, error) {
	// Validate request
	if req.GitHubURL == "" || req.Name == "" || req.Branch == "" {
		return nil, fmt.Errorf("github_url, name, and branch are required")
	}
	if req.PRDText == "" && len(req.PRDFile) == 0 {
		return nil, fmt.Errorf("either prd_text or prd_file must be provided")
	}

	// Extract PRD content
	prdContent := req.PRDText
	if prdContent == "" {
		prdContent = string(req.PRDFile)
	}

	// Generate project ID
	projectID := fmt.Sprintf("proj-%d", time.Now().Unix())

	// Create project directory in workspace
	projectPath := filepath.Join(bs.workspaceDir, projectID)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Clone or initialize repository
	if err := bs.cloneRepository(ctx, req.GitHubURL, req.Branch, projectPath); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Initialize project structure
	if err := bs.initializeProjectStructure(ctx, projectPath, prdContent); err != nil {
		return nil, fmt.Errorf("failed to initialize project structure: %w", err)
	}

	// Initialize beads
	if err := bs.initializeBeads(ctx, projectPath); err != nil {
		return nil, fmt.Errorf("failed to initialize beads: %w", err)
	}

	// Commit initial structure
	if err := bs.commitInitialStructure(ctx, projectPath); err != nil {
		return nil, fmt.Errorf("failed to commit initial structure: %w", err)
	}

	// Register project with Loom
	project, err := bs.projectManager.CreateProject(req.Name, projectPath, req.Branch, ".beads", map[string]string{
		"bootstrap":    "true",
		"github_url":   req.GitHubURL,
		"description": fmt.Sprintf("Bootstrapped project from PRD"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register project: %w", err)
	}

	// TODO: Create PM bead for PRD expansion
	// This will be done in a follow-up step after beads manager integration

	return &BootstrapResult{
		ProjectID:   project.ID,
		Status:      "initializing",
		InitialBead: "", // Will be populated when PM bead is created
	}, nil
}

// cloneRepository clones or initializes a git repository
func (bs *BootstrapService) cloneRepository(ctx context.Context, gitURL, branch, destPath string) error {
	// Try to clone the repository
	cmd := exec.CommandContext(ctx, "git", "clone", "--branch", branch, gitURL, destPath)
	if err := cmd.Run(); err != nil {
		// If clone fails, try to initialize and set remote
		cmd = exec.CommandContext(ctx, "git", "init", destPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}

		// Set remote
		cmd = exec.CommandContext(ctx, "git", "-C", destPath, "remote", "add", "origin", gitURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}

		// Create initial branch
		cmd = exec.CommandContext(ctx, "git", "-C", destPath, "checkout", "-b", branch)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
	}

	return nil
}

// initializeProjectStructure creates the project directory structure and template files
func (bs *BootstrapService) initializeProjectStructure(ctx context.Context, projectPath, prdContent string) error {
	// Create plans directory
	plansDir := filepath.Join(projectPath, "plans")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	// Write BOOTSTRAP.md with initial PRD
	bootstrapPath := filepath.Join(plansDir, "BOOTSTRAP.md")
	if err := os.WriteFile(bootstrapPath, []byte(prdContent), 0644); err != nil {
		return fmt.Errorf("failed to write BOOTSTRAP.md: %w", err)
	}

	// Copy template files
	if err := bs.copyTemplateFiles(projectPath); err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	return nil
}

// copyTemplateFiles copies template configuration files to the project
func (bs *BootstrapService) copyTemplateFiles(projectPath string) error {
	// Create settings.json from template
	settingsContent := `{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  },
  "workflowMode": "guided",
  "enableReviews": true
}
`
	settingsPath := filepath.Join(projectPath, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	// Create .mcp.json from template
	mcpContent := `{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  }
}
`
	mcpPath := filepath.Join(projectPath, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(mcpContent), 0644); err != nil {
		return fmt.Errorf("failed to write .mcp.json: %w", err)
	}

	return nil
}

// initializeBeads initializes the beads system for the project
func (bs *BootstrapService) initializeBeads(ctx context.Context, projectPath string) error {
	cmd := exec.CommandContext(ctx, "bd", "init")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run bd init: %w", err)
	}

	return nil
}

// commitInitialStructure commits the initial project structure
func (bs *BootstrapService) commitInitialStructure(ctx context.Context, projectPath string) error {
	// Stage all files
	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	// Commit
	commitMsg := "chore: initialize project with initial PRD\n\nBootstrapped by Loom\nCo-Authored-By: Loom <noreply@loom.dev>"
	cmd = exec.CommandContext(ctx, "git", "-C", projectPath, "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}
