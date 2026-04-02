// Package rcc provides a client for the Rocky Command Center (RCC) API.
// It enables loom agents to register with the RCC for fleet coordination,
// post heartbeats, and claim work items from the shared work queue.
//
// Configuration (via environment variables):
//   LOOM_RCC_URL          — RCC base URL (e.g. http://146.190.134.110:8789)
//   LOOM_RCC_AGENT_TOKEN  — Agent bearer token (rcc-agent-<name>-<hash>)
//   LOOM_RCC_AGENT_NAME   — Agent name for registration (default: "loom")
//   LOOM_RCC_HOST         — Host identifier (default: os.Hostname())
//   LOOM_RCC_ROLE         — Agent role (default: "autonomous-agent")
//
// Usage:
//   client := rcc.NewClient(rcc.ConfigFromEnv())
//   if err := client.Register(ctx); err != nil {
//       log.Warn("RCC registration failed (not fatal):", err)
//   }
//   defer client.StopHeartbeat()
package rcc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Config holds RCC client configuration.
type Config struct {
	BaseURL    string
	AgentToken string
	AgentName  string
	Host       string
	Role       string
}

// ConfigFromEnv builds a Config from environment variables.
func ConfigFromEnv() Config {
	host, _ := os.Hostname()
	cfg := Config{
		BaseURL:    os.Getenv("LOOM_RCC_URL"),
		AgentToken: os.Getenv("LOOM_RCC_AGENT_TOKEN"),
		AgentName:  os.Getenv("LOOM_RCC_AGENT_NAME"),
		Host:       host,
		Role:       os.Getenv("LOOM_RCC_ROLE"),
	}
	if cfg.AgentName == "" {
		cfg.AgentName = "loom"
	}
	if cfg.Role == "" {
		cfg.Role = "autonomous-agent"
	}
	return cfg
}

// IsConfigured returns true if the RCC URL and agent token are both set.
func (c Config) IsConfigured() bool {
	return c.BaseURL != "" && c.AgentToken != ""
}

// Client is an RCC API client.
type Client struct {
	cfg        Config
	httpClient *http.Client
	stopCh     chan struct{}
}

// NewClient creates a new RCC client with the given configuration.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

// IsAvailable returns true if the client is configured with a URL and token.
func (c *Client) IsAvailable() bool {
	return c.cfg.IsConfigured()
}

// post sends a POST request to the RCC API.
func (c *Client) post(ctx context.Context, path string, body interface{}) ([]byte, int, error) {
	b, _ := json.Marshal(body)
	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.AgentToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 32768))
	return respBytes, resp.StatusCode, nil
}

// get sends a GET request to the RCC API.
func (c *Client) get(ctx context.Context, path string) ([]byte, int, error) {
	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AgentToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 32768))
	return respBytes, resp.StatusCode, nil
}

// Register registers this loom instance with the RCC agent registry.
// Loom agents appear in the fleet dashboard and can receive work items.
func (c *Client) Register(ctx context.Context) error {
	if !c.IsAvailable() {
		return nil // not configured — skip silently
	}
	body := map[string]interface{}{
		"name": c.cfg.AgentName,
		"host": c.cfg.Host,
		"role": c.cfg.Role,
		"capabilities": []string{
			"autonomous-agent",
			"code-generation",
			"planning",
			"loom",
		},
		"metadata": map[string]string{
			"runtime": "loom",
			"version": os.Getenv("LOOM_VERSION"),
		},
	}
	respBytes, status, err := c.post(ctx, "/api/agents/register", body)
	if err != nil {
		return fmt.Errorf("RCC register POST failed: %w", err)
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("RCC register: HTTP %d: %s", status, string(respBytes))
	}
	return nil
}

// Heartbeat posts a single heartbeat to the RCC.
func (c *Client) Heartbeat(ctx context.Context, status string, note string) error {
	if !c.IsAvailable() {
		return nil
	}
	body := map[string]interface{}{
		"status": status,
		"host":   c.cfg.Host,
		"ts":     time.Now().UTC().Format(time.RFC3339),
		"note":   note,
	}
	_, _, err := c.post(ctx, "/api/heartbeat/"+c.cfg.AgentName, body)
	return err
}

// StartHeartbeat starts a background goroutine that posts heartbeats every
// interval to keep the loom agent visible in the RCC fleet dashboard.
func (c *Client) StartHeartbeat(interval time.Duration) {
	if !c.IsAvailable() {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = c.Heartbeat(ctx, "online", "loom running")
				cancel()
			}
		}
	}()
}

// StopHeartbeat stops the background heartbeat goroutine.
func (c *Client) StopHeartbeat() {
	select {
	case c.stopCh <- struct{}{}:
	default:
	}
}

// WorkQueueItem represents a single item from the RCC work queue.
type WorkQueueItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Assignee    string `json:"assignee"`
	Status      string `json:"status"`
	Tags        []string `json:"tags"`
}

// FetchWorkItems fetches pending work items from the RCC queue that are
// assigned to this agent or to "all".
func (c *Client) FetchWorkItems(ctx context.Context) ([]WorkQueueItem, error) {
	if !c.IsAvailable() {
		return nil, nil
	}
	respBytes, status, err := c.get(ctx, "/api/queue")
	if err != nil {
		return nil, fmt.Errorf("RCC queue fetch failed: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("RCC queue: HTTP %d", status)
	}
	var items []WorkQueueItem
	if err := json.Unmarshal(respBytes, &items); err != nil {
		// Try wrapped format
		var resp struct {
			Items []WorkQueueItem `json:"items"`
		}
		if err2 := json.Unmarshal(respBytes, &resp); err2 != nil {
			return nil, fmt.Errorf("RCC queue parse failed: %w", err)
		}
		items = resp.Items
	}
	// Filter to items assigned to us or to "all"
	agentName := c.cfg.AgentName
	var mine []WorkQueueItem
	for _, item := range items {
		if item.Status == "pending" &&
			(item.Assignee == agentName || item.Assignee == "all" || item.Assignee == "any") {
			mine = append(mine, item)
		}
	}
	return mine, nil
}

// ClaimItem attempts to claim a work item for this agent.
// Returns true if the claim succeeded.
func (c *Client) ClaimItem(ctx context.Context, itemID string) (bool, error) {
	if !c.IsAvailable() {
		return false, nil
	}
	body := map[string]interface{}{
		"status":    "in-progress",
		"claimedBy": c.cfg.AgentName,
		"claimedAt": time.Now().UTC().Format(time.RFC3339),
		"_author":   c.cfg.AgentName,
	}
	respBytes, status, err := c.post(ctx, "/api/item/"+itemID, body)
	if err != nil {
		return false, err
	}
	if status == 401 || status == 403 {
		return false, nil // not authorized — item owned by another agent
	}
	if status != 200 {
		return false, fmt.Errorf("claim HTTP %d: %s", status, string(respBytes))
	}
	var resp struct{ OK bool `json:"ok"` }
	_ = json.Unmarshal(respBytes, &resp)
	return resp.OK, nil
}

// CompleteItem marks a work item as completed with a result string.
func (c *Client) CompleteItem(ctx context.Context, itemID, result string) error {
	if !c.IsAvailable() {
		return nil
	}
	body := map[string]interface{}{
		"status":      "completed",
		"claimedBy":   c.cfg.AgentName,
		"completedAt": time.Now().UTC().Format(time.RFC3339),
		"result":      result,
		"_author":     c.cfg.AgentName,
	}
	_, status, err := c.post(ctx, "/api/item/"+itemID, body)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("complete HTTP %d", status)
	}
	return nil
}
