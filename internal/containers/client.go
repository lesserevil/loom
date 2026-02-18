package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Helper functions for converting map[string]interface{} to TaskRequest
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if subMap, ok := v.(map[string]interface{}); ok {
			return subMap
		}
	}
	return nil
}

// ProjectAgentClient communicates with project agent containers
type ProjectAgentClient struct {
	baseURL    string
	projectID  string
	httpClient *http.Client
}

// TaskRequest represents a task to send to project agent
type TaskRequest struct {
	TaskID    string                 `json:"task_id"`
	BeadID    string                 `json:"bead_id"`
	Action    string                 `json:"action"`
	ProjectID string                 `json:"project_id"`
	Params    map[string]interface{} `json:"params"`
}

// TaskResult represents result from project agent
type TaskResult struct {
	TaskID   string                 `json:"task_id"`
	BeadID   string                 `json:"bead_id"`
	Success  bool                   `json:"success"`
	Output   string                 `json:"output"`
	Error    string                 `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AgentStatus represents project agent status
type AgentStatus struct {
	ProjectID   string                 `json:"project_id"`
	WorkDir     string                 `json:"work_dir"`
	Busy        bool                   `json:"busy"`
	CurrentTask map[string]interface{} `json:"current_task,omitempty"`
}

// NewProjectAgentClient creates a new project agent client
func NewProjectAgentClient(baseURL, projectID string) *ProjectAgentClient {
	return &ProjectAgentClient{
		baseURL:   baseURL,
		projectID: projectID,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Health checks if the project agent is healthy
func (c *ProjectAgentClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// Status returns the current status of the project agent
func (c *ProjectAgentClient) Status(ctx context.Context) (*AgentStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status request failed: %d - %s", resp.StatusCode, body)
	}

	var status AgentStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// ExecuteTask sends a task to the project agent for execution
func (c *ProjectAgentClient) ExecuteTask(ctx context.Context, req interface{}) error {
	// Convert interface{} to TaskRequest if needed
	var taskReq *TaskRequest
	switch r := req.(type) {
	case *TaskRequest:
		taskReq = r
	case map[string]interface{}:
		// Convert map to TaskRequest
		taskReq = &TaskRequest{
			TaskID:    getStringFromMap(r, "task_id"),
			BeadID:    getStringFromMap(r, "bead_id"),
			Action:    getStringFromMap(r, "action"),
			ProjectID: getStringFromMap(r, "project_id"),
			Params:    getMapFromMap(r, "params"),
		}
	default:
		return fmt.Errorf("unsupported request type: %T", req)
	}

	// Ensure project ID matches
	taskReq.ProjectID = c.projectID

	body, err := json.Marshal(taskReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/task", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("task submission failed: %d - %s", resp.StatusCode, respBody)
	}

	return nil
}

// ExecuteTaskSync sends a task and waits for the result (blocking)
// This is a convenience method - in production, use ExecuteTask + result webhook
func (c *ProjectAgentClient) ExecuteTaskSync(ctx context.Context, req *TaskRequest, timeout time.Duration) (*TaskResult, error) {
	// Send task
	if err := c.ExecuteTask(ctx, req); err != nil {
		return nil, err
	}

	// Poll for completion (simplified - in production use webhooks)
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("task execution timeout")
			}

			// Check if agent is still busy with this task
			status, err := c.Status(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to check status: %w", err)
			}

			// If not busy, task completed (we'd need proper result retrieval)
			// For now, return success if agent is idle
			if !status.Busy {
				// TODO: Implement proper result retrieval endpoint
				return &TaskResult{
					TaskID:  req.TaskID,
					BeadID:  req.BeadID,
					Success: true,
					Output:  "Task completed (result retrieval TBD)",
				}, nil
			}
		}
	}
}
