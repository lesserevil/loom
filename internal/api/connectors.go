package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jordanhubbard/loom/pkg/connectors"
)

// ConnectorResponse is the API response format for a connector
type ConnectorResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        connectors.ConnectorType `json:"type"`
	Mode        connectors.ConnectionMode `json:"mode"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
	Endpoint    string                 `json:"endpoint"`
	Status      connectors.ConnectorStatus `json:"status"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// HandleConnectors routes connector requests based on path and method
func (s *Server) HandleConnectors(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/v1/connectors
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/connectors")

	// Route: /api/v1/connectors or /api/v1/connectors/
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			s.listConnectors(w, r)
		case http.MethodPost:
			s.createConnector(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Route: /api/v1/connectors/health
	if path == "/health" {
		if r.Method == http.MethodGet {
			s.checkAllHealth(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Route: /api/v1/connectors/{id}* - extract ID
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Connector ID required", http.StatusBadRequest)
		return
	}

	id := parts[0]

	// Check for sub-paths
	if len(parts) > 1 {
		switch parts[1] {
		case "health":
			if r.Method == http.MethodGet {
				s.checkConnectorHealth(w, r, id)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "test":
			if r.Method == http.MethodPost {
				s.testConnector(w, r, id)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.Error(w, "Unknown endpoint", http.StatusNotFound)
		}
		return
	}

	// Route: /api/v1/connectors/{id}
	switch r.Method {
	case http.MethodGet:
		s.getConnector(w, r, id)
	case http.MethodPut:
		s.updateConnector(w, r, id)
	case http.MethodDelete:
		s.deleteConnector(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listConnectors returns all registered connectors
func (s *Server) listConnectors(w http.ResponseWriter, r *http.Request) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	allConnectors := s.app.GetConnectorManager().ListConnectors()
	healthStatus := s.app.GetConnectorManager().GetHealthStatus()

	responses := make([]ConnectorResponse, 0, len(allConnectors))
	for _, c := range allConnectors {
		cfg := c.GetConfig()
		status := healthStatus[c.ID()]
		if status == "" {
			status = connectors.ConnectorStatusUnknown
		}

		responses = append(responses, ConnectorResponse{
			ID:          c.ID(),
			Name:        c.Name(),
			Type:        c.Type(),
			Mode:        cfg.Mode,
			Enabled:     cfg.Enabled,
			Description: c.Description(),
			Endpoint:    c.GetEndpoint(),
			Status:      status,
			Tags:        cfg.Tags,
			Metadata:    cfg.Metadata,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connectors": responses,
		"count":      len(responses),
	})
}

// getConnector returns a specific connector by ID
func (s *Server) getConnector(w http.ResponseWriter, r *http.Request, id string) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	connector, err := s.app.GetConnectorManager().GetConnector(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cfg := connector.GetConfig()
	status := s.app.GetConnectorManager().GetConnectorHealth(id)

	response := ConnectorResponse{
		ID:          connector.ID(),
		Name:        connector.Name(),
		Type:        connector.Type(),
		Mode:        cfg.Mode,
		Enabled:     cfg.Enabled,
		Description: connector.Description(),
		Endpoint:    connector.GetEndpoint(),
		Status:      status,
		Tags:        cfg.Tags,
		Metadata:    cfg.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createConnector adds a new connector
func (s *Server) createConnector(w http.ResponseWriter, r *http.Request) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}
	var cfg connectors.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if cfg.ID == "" {
		http.Error(w, "Connector ID is required", http.StatusBadRequest)
		return
	}
	if cfg.Name == "" {
		http.Error(w, "Connector name is required", http.StatusBadRequest)
		return
	}
	if cfg.Host == "" {
		http.Error(w, "Connector host is required", http.StatusBadRequest)
		return
	}
	if cfg.Port == 0 {
		http.Error(w, "Connector port is required", http.StatusBadRequest)
		return
	}

	// Add the connector
	if err := s.app.GetConnectorManager().AddConnector(cfg); err != nil {
		http.Error(w, "Failed to create connector: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save configuration
	if err := s.app.GetConnectorManager().SaveConfig(); err != nil {
		http.Error(w, "Failed to save configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Connector created successfully",
		"id":      cfg.ID,
	})
}

// updateConnector updates an existing connector
func (s *Server) updateConnector(w http.ResponseWriter, r *http.Request, id string) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	var cfg connectors.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure ID matches
	cfg.ID = id

	// Update the connector
	if err := s.app.GetConnectorManager().UpdateConnector(id, cfg); err != nil {
		http.Error(w, "Failed to update connector: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Connector updated successfully",
	})
}

// deleteConnector removes a connector
func (s *Server) deleteConnector(w http.ResponseWriter, r *http.Request, id string) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	if err := s.app.GetConnectorManager().RemoveConnector(id); err != nil {
		http.Error(w, "Failed to delete connector: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Connector deleted successfully",
	})
}

// checkConnectorHealth checks health of a specific connector
func (s *Server) checkConnectorHealth(w http.ResponseWriter, r *http.Request, id string) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	connector, err := s.app.GetConnectorManager().GetConnector(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	status, err := connector.HealthCheck(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      id,
			"status":  status,
			"healthy": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"status":  status,
		"healthy": status == connectors.ConnectorStatusHealthy,
	})
}

// checkAllHealth checks health of all connectors
func (s *Server) checkAllHealth(w http.ResponseWriter, r *http.Request) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}
	healthStatus := s.app.GetConnectorManager().GetHealthStatus()

	results := make(map[string]interface{})
	for id, status := range healthStatus {
		results[id] = map[string]interface{}{
			"status":  status,
			"healthy": status == connectors.ConnectorStatusHealthy,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

// testConnector tests connectivity to a connector
func (s *Server) testConnector(w http.ResponseWriter, r *http.Request, id string) {
	if s.app.GetConnectorManager() == nil {
		http.Error(w, "Connector manager not initialized", http.StatusInternalServerError)
		return
	}

	connector, err := s.app.GetConnectorManager().GetConnector(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Test connection with health check
	status, err := connector.HealthCheck(r.Context())

	response := map[string]interface{}{
		"id":       id,
		"status":   status,
		"endpoint": connector.GetEndpoint(),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["success"] = status == connectors.ConnectorStatusHealthy
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
