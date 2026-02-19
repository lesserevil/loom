package containers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// AgentClient interface for executing tasks in project containers
type AgentClient interface {
	ExecuteTask(ctx context.Context, req interface{}) error
	Health(ctx context.Context) error
	Status(ctx context.Context) (*AgentStatus, error)
}

// Orchestrator manages project container lifecycle
type Orchestrator struct {
	projectsRoot    string
	composeFile     string
	projectAgents   map[string]*ProjectAgentClient
	mu              sync.RWMutex
	controlPlaneURL string
	messageBus      MessageBus // NATS message bus for async task publishing
}

// NewOrchestrator creates a new container orchestrator
func NewOrchestrator(projectsRoot, controlPlaneURL string) (*Orchestrator, error) {
	composeFile := filepath.Join(projectsRoot, "docker-compose-projects.yml")

	return &Orchestrator{
		projectsRoot:    projectsRoot,
		composeFile:     composeFile,
		projectAgents:   make(map[string]*ProjectAgentClient),
		controlPlaneURL: controlPlaneURL,
	}, nil
}

// SetMessageBus sets the NATS message bus for async task publishing
func (o *Orchestrator) SetMessageBus(mb MessageBus) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.messageBus = mb
	log.Printf("[Orchestrator] Message bus configured for container orchestration")
}

// EnsureProjectContainer ensures a project container is running
func (o *Orchestrator) EnsureProjectContainer(ctx context.Context, project *models.Project) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if already running
	if agent, exists := o.projectAgents[project.ID]; exists {
		// Verify it's healthy
		if err := agent.Health(ctx); err == nil {
			log.Printf("[Containers] Project %s container already healthy", project.ID)
			return nil
		}
		// Unhealthy - remove and recreate
		log.Printf("[Containers] Project %s container unhealthy, recreating", project.ID)
		delete(o.projectAgents, project.ID)
	}

	// Generate docker-compose.yml for this project
	if err := o.generateComposeFile(project); err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	// Start container using docker-compose
	if err := o.startContainer(ctx, project); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to be healthy
	if err := o.waitForHealth(ctx, project, 60*time.Second); err != nil {
		return fmt.Errorf("container failed to become healthy: %w", err)
	}

	// Create agent client
	agentURL := fmt.Sprintf("http://loom-project-%s:8090", project.ID)
	agent := NewProjectAgentClient(agentURL, project.ID)

	// Inject message bus if available
	if o.messageBus != nil {
		agent.SetMessageBus(o.messageBus)
		log.Printf("[Containers] Project %s agent configured with NATS message bus", project.ID)
	}

	o.projectAgents[project.ID] = agent

	log.Printf("[Containers] Project %s container started and healthy", project.ID)
	return nil
}

// GetAgent returns the agent client for a project
func (o *Orchestrator) GetAgent(projectID string) (AgentClient, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.projectAgents[projectID]
	if !exists {
		return nil, fmt.Errorf("no agent for project %s", projectID)
	}

	return agent, nil
}

// StopProjectContainer stops a project's container
func (o *Orchestrator) StopProjectContainer(ctx context.Context, projectID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	serviceName := fmt.Sprintf("loom-project-%s", projectID)

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", o.composeFile, "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %s - %w", output, err)
	}

	delete(o.projectAgents, projectID)
	log.Printf("[Containers] Stopped project %s container", projectID)
	return nil
}

// generateComposeFile generates docker-compose.yml for project containers
func (o *Orchestrator) generateComposeFile(project *models.Project) error {
	// Read all existing projects to generate complete compose file
	// For now, generate for single project (extend later for multiple)

	tmpl := `version: '3.8'

services:
  loom-project-{{.ProjectID}}:
    image: loom-project:{{.ProjectID}}
    container_name: loom-project-{{.ProjectID}}
    build:
      context: .
      dockerfile: {{.Dockerfile}}
      args:
        PROJECT_ID: "{{.ProjectID}}"
    environment:
      - PROJECT_ID={{.ProjectID}}
      - CONTROL_PLANE_URL={{.ControlPlaneURL}}
      - WORK_DIR=/workspace
      - GITLAB_TOKEN=${GITLAB_TOKEN}
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    volumes:
      # Isolated workspace - NO host mounts to prevent root filesystem contamination
      - loom-project-{{.ProjectID}}-workspace:/workspace
      # SSH keys for git (read-only)
      - {{.ProjectsRoot}}/{{.ProjectID}}/keys:/root/.ssh:ro
    networks:
      - loom_loom-network
    restart: unless-stopped
    cap_add:
      - SYS_ADMIN  # For hermetic operations
    security_opt:
      - apparmor:unconfined  # Allow full root capabilities in isolated container

networks:
  loom_loom-network:
    external: true

volumes:
  loom-project-{{.ProjectID}}-workspace:
    driver: local
`

	t, err := template.New("compose").Parse(tmpl)
	if err != nil {
		return err
	}

	// Determine Dockerfile path (create default if needed)
	dockerfilePath := filepath.Join(o.projectsRoot, project.ID, "Dockerfile.project")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		// Generate default Dockerfile
		if err := o.generateDefaultDockerfile(project, dockerfilePath); err != nil {
			return fmt.Errorf("failed to generate dockerfile: %w", err)
		}
	}

	data := map[string]string{
		"ProjectID":        project.ID,
		"ControlPlaneURL":  o.controlPlaneURL,
		"Dockerfile":       dockerfilePath,
		"ProjectsRoot":     o.projectsRoot,
	}

	f, err := os.Create(o.composeFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// generateDefaultDockerfile creates a default Dockerfile for project containers
func (o *Orchestrator) generateDefaultDockerfile(project *models.Project, path string) error {
	// Determine base image based on project type
	baseImage := "ubuntu:22.04"
	if project.Context != nil {
		if img, ok := project.Context["base_image"]; ok {
			baseImage = img
		}
	}

	dockerfile := fmt.Sprintf(`# Auto-generated Dockerfile for project: %s
FROM %s

# Install essential tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    ca-certificates \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Go (common for many projects) - detect architecture at build time
RUN ARCH=$(uname -m) && \
    case "$ARCH" in \
        x86_64) GOARCH=amd64 ;; \
        aarch64|arm64) GOARCH=arm64 ;; \
        armv7l) GOARCH=armv6l ;; \
        *) echo "Unsupported arch: $ARCH" && exit 1 ;; \
    esac && \
    wget https://go.dev/dl/go1.25.7.linux-${GOARCH}.tar.gz && \
    tar -C /usr/local -xzf go1.25.7.linux-${GOARCH}.tar.gz && \
    rm go1.25.7.linux-${GOARCH}.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/root/go"

# Create workspace
WORKDIR /workspace

# Copy project agent binary
COPY --from=loom:latest /app/loom-project-agent /usr/local/bin/loom-project-agent
RUN chmod +x /usr/local/bin/loom-project-agent

# Git config
RUN git config --global user.name "Loom Agent" && \
    git config --global user.email "loom@localhost"

# Entrypoint runs project agent
ENTRYPOINT ["/usr/local/bin/loom-project-agent"]
`, project.Name, baseImage)

	return os.WriteFile(path, []byte(dockerfile), 0644)
}

// startContainer starts a project container using docker-compose
func (o *Orchestrator) startContainer(ctx context.Context, project *models.Project) error {
	serviceName := fmt.Sprintf("loom-project-%s", project.ID)

	// Build the container image first
	buildCmd := exec.CommandContext(ctx, "docker", "compose", "-f", o.composeFile, "build", serviceName)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	// Start the container
	startCmd := exec.CommandContext(ctx, "docker", "compose", "-f", o.composeFile, "up", "-d", serviceName)
	output, err := startCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker up failed: %s - %w", output, err)
	}

	log.Printf("[Containers] Started container for project %s", project.ID)
	return nil
}

// waitForHealth waits for a container to become healthy
func (o *Orchestrator) waitForHealth(ctx context.Context, project *models.Project, timeout time.Duration) error {
	agentURL := fmt.Sprintf("http://loom-project-%s:8090", project.ID)
	agent := NewProjectAgentClient(agentURL, project.ID)

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for container health")
			}

			if err := agent.Health(ctx); err == nil {
				return nil
			}
			log.Printf("[Containers] Waiting for project %s container to be healthy...", project.ID)
		}
	}
}

// ListRunningContainers returns list of running project containers
func (o *Orchestrator) ListRunningContainers(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=loom-project-", "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if line != "" {
			// Extract project ID from container name (loom-project-XXX)
			parts := strings.SplitN(line, "loom-project-", 2)
			if len(parts) == 2 {
				containers = append(containers, parts[1])
			}
		}
	}

	return containers, nil
}

// StopAll stops all project containers
func (o *Orchestrator) StopAll(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, err := os.Stat(o.composeFile); os.IsNotExist(err) {
		return nil // No compose file, nothing to stop
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", o.composeFile, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop containers: %s - %w", output, err)
	}

	o.projectAgents = make(map[string]*ProjectAgentClient)
	log.Println("[Containers] Stopped all project containers")
	return nil
}
