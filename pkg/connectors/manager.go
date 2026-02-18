package connectors

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager handles connector lifecycle, configuration, and health monitoring
type Manager struct {
	registry      *Registry
	configPath    string
	healthTicker  *time.Ticker
	healthResults map[string]ConnectorStatus
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new connector manager
func NewManager(configPath string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		registry:      NewRegistry(),
		configPath:    configPath,
		healthResults: make(map[string]ConnectorStatus),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// LoadConfig loads connector configurations from YAML file
func (m *Manager) LoadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, create default configuration
			return m.CreateDefaultConfig()
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config ConnectorConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Initialize connectors from config
	for _, cfg := range config.Connectors {
		if err := m.AddConnector(cfg); err != nil {
			log.Printf("Warning: Failed to initialize connector %s: %v", cfg.ID, err)
		}
	}

	return nil
}

// ConnectorConfigFile represents the connectors.yaml file structure
type ConnectorConfigFile struct {
	Connectors []Config `yaml:"connectors"`
}

// CreateDefaultConfig creates a default configuration with built-in connectors
func (m *Manager) CreateDefaultConfig() error {
	defaultConfig := ConnectorConfigFile{
		Connectors: []Config{
			{
				ID:          "prometheus",
				Name:        "Prometheus",
				Type:        ConnectorTypeObservability,
				Mode:        ConnectionModeLocal,
				Enabled:     true,
				Description: "Metrics collection and monitoring",
				Host:        "prometheus",
				Port:        9090,
				Scheme:      "http",
				HealthCheck: &HealthCheckConfig{
					Enabled:  true,
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Path:     "/-/healthy",
				},
				Tags: []string{"observability", "metrics", "built-in"},
			},
			{
				ID:          "grafana",
				Name:        "Grafana",
				Type:        ConnectorTypeObservability,
				Mode:        ConnectionModeLocal,
				Enabled:     true,
				Description: "Metrics visualization and dashboards",
				Host:        "grafana",
				Port:        3000,
				Scheme:      "http",
				HealthCheck: &HealthCheckConfig{
					Enabled:  true,
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Path:     "/api/health",
				},
				Tags: []string{"observability", "visualization", "built-in"},
			},
			{
				ID:          "jaeger",
				Name:        "Jaeger",
				Type:        ConnectorTypeObservability,
				Mode:        ConnectionModeLocal,
				Enabled:     true,
				Description: "Distributed tracing",
				Host:        "jaeger",
				Port:        16686,
				Scheme:      "http",
				HealthCheck: &HealthCheckConfig{
					Enabled:  true,
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
				},
				Tags: []string{"observability", "tracing", "built-in"},
			},
		},
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write default config
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	log.Printf("Created default connector configuration at %s", m.configPath)

	// Initialize default connectors
	for _, cfg := range defaultConfig.Connectors {
		if err := m.AddConnector(cfg); err != nil {
			log.Printf("Warning: Failed to initialize connector %s: %v", cfg.ID, err)
		}
	}

	return nil
}

// SaveConfig persists current connector configurations
func (m *Manager) SaveConfig() error {
	connectors := m.registry.List()
	configs := make([]Config, 0, len(connectors))
	for _, c := range connectors {
		configs = append(configs, c.GetConfig())
	}

	configFile := ConnectorConfigFile{
		Connectors: configs,
	}

	data, err := yaml.Marshal(configFile)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddConnector creates and registers a connector from configuration
func (m *Manager) AddConnector(cfg Config) error {
	var connector Connector

	// Factory pattern based on connector type and ID
	switch cfg.Type {
	case ConnectorTypeObservability:
		switch cfg.ID {
		case "prometheus":
			connector = NewPrometheusConnector(cfg)
		case "grafana":
			connector = NewGrafanaConnector(cfg)
		case "jaeger":
			connector = NewJaegerConnector(cfg)
		default:
			return fmt.Errorf("unknown observability connector: %s", cfg.ID)
		}
	case ConnectorTypeAgent:
		if cfg.ID == "openclaw" || cfg.Metadata["type"] == "openclaw" {
			connector = NewOpenClawConnector(cfg)
		} else {
			return fmt.Errorf("unknown agent connector: %s", cfg.ID)
		}
	default:
		return fmt.Errorf("unsupported connector type: %s", cfg.Type)
	}

	// Initialize the connector
	if err := connector.Initialize(m.ctx, cfg); err != nil {
		return fmt.Errorf("failed to initialize connector: %w", err)
	}

	// Register in registry
	if err := m.registry.Register(connector); err != nil {
		return fmt.Errorf("failed to register connector: %w", err)
	}

	log.Printf("Registered connector: %s (%s) at %s", cfg.Name, cfg.ID, cfg.GetFullURL())
	return nil
}

// RemoveConnector removes a connector by ID
func (m *Manager) RemoveConnector(id string) error {
	if err := m.registry.Remove(id); err != nil {
		return err
	}

	// Save updated configuration
	return m.SaveConfig()
}

// GetConnector retrieves a connector by ID
func (m *Manager) GetConnector(id string) (Connector, error) {
	return m.registry.Get(id)
}

// ListConnectors returns all registered connectors
func (m *Manager) ListConnectors() []Connector {
	return m.registry.List()
}

// ListConnectorsByType returns connectors of a specific type
func (m *Manager) ListConnectorsByType(connectorType ConnectorType) []Connector {
	return m.registry.ListByType(connectorType)
}

// StartHealthMonitoring begins periodic health checks for all connectors
func (m *Manager) StartHealthMonitoring(interval time.Duration) {
	m.healthTicker = time.NewTicker(interval)

	go func() {
		// Do initial health check
		m.checkAllHealth()

		for {
			select {
			case <-m.healthTicker.C:
				m.checkAllHealth()
			case <-m.ctx.Done():
				return
			}
		}
	}()

	log.Printf("Started connector health monitoring (interval: %v)", interval)
}

func (m *Manager) checkAllHealth() {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	results := m.registry.HealthCheckAll(ctx)

	m.mu.Lock()
	m.healthResults = results
	m.mu.Unlock()

	// Log unhealthy connectors
	for id, status := range results {
		if status != ConnectorStatusHealthy {
			log.Printf("Connector %s is %s", id, status)
		}
	}
}

// GetHealthStatus returns the current health status of all connectors
func (m *Manager) GetHealthStatus() map[string]ConnectorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	results := make(map[string]ConnectorStatus, len(m.healthResults))
	for k, v := range m.healthResults {
		results[k] = v
	}
	return results
}

// GetConnectorHealth returns the health status of a specific connector
func (m *Manager) GetConnectorHealth(id string) ConnectorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, exists := m.healthResults[id]; exists {
		return status
	}
	return ConnectorStatusUnknown
}

// UpdateConnector updates an existing connector's configuration
func (m *Manager) UpdateConnector(id string, cfg Config) error {
	// Remove old connector
	if err := m.registry.Remove(id); err != nil {
		return fmt.Errorf("failed to remove old connector: %w", err)
	}

	// Add new connector with updated config
	if err := m.AddConnector(cfg); err != nil {
		return fmt.Errorf("failed to add updated connector: %w", err)
	}

	// Save configuration
	return m.SaveConfig()
}

// Close shuts down the connector manager
func (m *Manager) Close() error {
	m.cancel()

	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}

	// Close all connectors
	connectors := m.registry.List()
	for _, c := range connectors {
		if err := c.Close(); err != nil {
			log.Printf("Error closing connector %s: %v", c.ID(), err)
		}
	}

	return nil
}
