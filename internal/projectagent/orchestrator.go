package projectagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// DefaultRoles is the full set of agent roles run inside a project container.
var DefaultRoles = []string{"coder", "reviewer", "qa", "pm", "architect"}

// OrchestratorConfig holds configuration for the in-container multi-role orchestrator.
type OrchestratorConfig struct {
	ProjectID         string
	ControlPlaneURL   string
	NatsURL           string
	WorkDir           string
	HeartbeatInterval time.Duration

	// Service identity (injected from env by container orchestrator)
	ServiceID  string
	InstanceID string

	// LLM provider (shared across all roles)
	ProviderEndpoint string
	ProviderModel    string
	ProviderAPIKey   string

	// Persona base path — per-role persona loaded as {PersonaBasePath}/{role}.md
	PersonaBasePath string

	// Roles to run; defaults to DefaultRoles if empty
	Roles []string

	// Action loop settings (applied to all roles)
	ActionLoopEnabled bool
	MaxLoopIterations int
}

// InContainerOrchestrator manages multiple role-based agents within a single project container.
// It runs all roles as concurrent goroutines sharing one NATS connection and one HTTP server.
type InContainerOrchestrator struct {
	cfg    OrchestratorConfig
	agents map[string]*Agent // role -> Agent
	mu     sync.RWMutex
}

// NewInContainerOrchestrator creates a new multi-role orchestrator.
func NewInContainerOrchestrator(cfg OrchestratorConfig) *InContainerOrchestrator {
	if len(cfg.Roles) == 0 {
		cfg.Roles = DefaultRoles
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	if cfg.MaxLoopIterations == 0 {
		cfg.MaxLoopIterations = 20
	}
	return &InContainerOrchestrator{
		cfg:    cfg,
		agents: make(map[string]*Agent),
	}
}

// Start initialises all role agents and begins serving HTTP on port 8090.
// Blocks until ctx is cancelled.
func (o *InContainerOrchestrator) Start(ctx context.Context) error {
	log.Printf("[Orchestrator] Starting multi-role agent container for project %s (roles: %v)",
		o.cfg.ProjectID, o.cfg.Roles)

	// Build one Agent per role. They share project/NATS config but each has
	// its own role, persona, NATS subscription, and result channel.
	for _, role := range o.cfg.Roles {
		personaPath := ""
		if o.cfg.PersonaBasePath != "" {
			personaPath = fmt.Sprintf("%s/%s.md", o.cfg.PersonaBasePath, role)
		}

		ag, err := New(Config{
			ProjectID:         o.cfg.ProjectID,
			ControlPlaneURL:   o.cfg.ControlPlaneURL,
			WorkDir:           o.cfg.WorkDir,
			HeartbeatInterval: o.cfg.HeartbeatInterval,
			NatsURL:           o.cfg.NatsURL,
			ServiceID:         fmt.Sprintf("%s-%s", o.cfg.ServiceID, role),
			InstanceID:        fmt.Sprintf("%s-%s", o.cfg.InstanceID, role),
			Role:              role,
			ProviderEndpoint:  o.cfg.ProviderEndpoint,
			ProviderModel:     o.cfg.ProviderModel,
			ProviderAPIKey:    o.cfg.ProviderAPIKey,
			PersonaPath:       personaPath,
			ActionLoopEnabled: o.cfg.ActionLoopEnabled,
			MaxLoopIterations: o.cfg.MaxLoopIterations,
		})
		if err != nil {
			return fmt.Errorf("failed to create agent for role %s: %w", role, err)
		}

		o.mu.Lock()
		o.agents[role] = ag
		o.mu.Unlock()
	}

	// Start all agents concurrently.
	var wg sync.WaitGroup
	for role, ag := range o.agents {
		wg.Add(1)
		go func(r string, a *Agent) {
			defer wg.Done()
			if err := a.Start(ctx); err != nil && err != context.Canceled {
				log.Printf("[Orchestrator] Agent %s exited with error: %v", r, err)
			}
		}(role, ag)
	}

	// Register aggregated HTTP handlers.
	mux := http.NewServeMux()
	o.registerHandlers(mux)

	server := &http.Server{
		Addr:    ":8090",
		Handler: mux,
	}

	go func() {
		log.Printf("[Orchestrator] HTTP server listening on :8090 (project=%s, roles=%v)",
			o.cfg.ProjectID, o.cfg.Roles)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Orchestrator] HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	wg.Wait()
	log.Printf("[Orchestrator] All agents stopped for project %s", o.cfg.ProjectID)
	return nil
}

// registerHandlers wires up the aggregated HTTP endpoints.
func (o *InContainerOrchestrator) registerHandlers(mux *http.ServeMux) {
	// /health — aggregate health of all role agents
	mux.HandleFunc("/health", o.handleHealth)

	// /status — per-role status snapshot
	mux.HandleFunc("/status", o.handleStatus)

	// /results/ — delegate to the matching role agent by looking up the taskID in each store
	mux.HandleFunc("/results/", o.handleResults)

	// Per-role delegate endpoints: all other requests go to the primary (coder) agent
	// to preserve the existing /task, /exec, /files/*, /git/* contract.
	primary := o.primaryAgent()
	if primary != nil {
		primary.RegisterHandlers(mux)
	}
}

// primaryAgent returns the coder agent as the primary handler for generic endpoints.
func (o *InContainerOrchestrator) primaryAgent() *Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if ag, ok := o.agents["coder"]; ok {
		return ag
	}
	// Fallback: first agent in map.
	for _, ag := range o.agents {
		return ag
	}
	return nil
}

func (o *InContainerOrchestrator) handleHealth(w http.ResponseWriter, r *http.Request) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	type roleHealth struct {
		Role   string `json:"role"`
		Busy   bool   `json:"busy"`
		Status string `json:"status"`
	}

	roles := make([]roleHealth, 0, len(o.agents))
	for role, ag := range o.agents {
		roles = append(roles, roleHealth{
			Role:   role,
			Busy:   ag.currentTask != nil,
			Status: "online",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"project_id": o.cfg.ProjectID,
		"roles":      roles,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func (o *InContainerOrchestrator) handleStatus(w http.ResponseWriter, r *http.Request) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	type roleStatus struct {
		Role        string      `json:"role"`
		Busy        bool        `json:"busy"`
		CurrentTask interface{} `json:"current_task,omitempty"`
	}

	roles := make([]roleStatus, 0, len(o.agents))
	for role, ag := range o.agents {
		rs := roleStatus{Role: role, Busy: ag.currentTask != nil}
		if ag.currentTask != nil {
			rs.CurrentTask = map[string]interface{}{
				"task_id":  ag.currentTask.Request.TaskID,
				"bead_id":  ag.currentTask.Request.BeadID,
				"action":   ag.currentTask.Request.Action,
				"duration": time.Since(ag.currentTask.StartTime).String(),
			}
		}
		roles = append(roles, rs)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id": o.cfg.ProjectID,
		"roles":      roles,
		"work_dir":   o.cfg.WorkDir,
	})
}

func (o *InContainerOrchestrator) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := r.URL.Path[len("/results/"):]
	if taskID == "" {
		http.Error(w, "task_id required", http.StatusBadRequest)
		return
	}

	// Search all role agents for the result.
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, ag := range o.agents {
		if val, ok := ag.resultStore.Load(taskID); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(val)
			return
		}
	}
	http.Error(w, "result not found", http.StatusNotFound)
}
