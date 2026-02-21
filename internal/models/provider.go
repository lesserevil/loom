package models

import (
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// Provider represents an AI engine running on-prem or in the cloud.
// With TokenHub as the sole provider, this struct is kept minimal —
// just enough for bootstrap registration and heartbeat monitoring.
type Provider struct {
	models.EntityMetadata `json:",inline"`

	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`     // always "openai" (TokenHub speaks OpenAI-compatible API)
	Endpoint        string   `json:"endpoint"` // URL to the provider
	Model           string   `json:"model"`    // Active model for this provider
	ConfiguredModel string   `json:"configured_model"`
	SelectedModel   string   `json:"selected_model"`
	Description     string   `json:"description"`
	RequiresKey     bool     `json:"requires_key"`      // Whether this provider needs API credentials
	KeyID           string   `json:"key_id"`            // Reference to encrypted key in key manager
	APIKey          string   `json:"api_key,omitempty"` // Plaintext API key (persisted encrypted-at-rest via DB)
	OwnerID         string   `json:"owner_id"`          // User ID who owns this provider (for multi-tenant)
	IsShared        bool     `json:"is_shared"`         // If true, provider available to all users
	Status          string   `json:"status"`            // active, inactive, healthy, failed
	ContextWindow   int      `json:"context_window"`    // Maximum context window size
	Tags            []string `json:"tags"`              // Custom tags for filtering

	LastHeartbeatAt        time.Time `json:"last_heartbeat_at"`
	LastHeartbeatLatencyMs int64     `json:"last_heartbeat_latency_ms"`
	LastHeartbeatError     string    `json:"last_heartbeat_error"`

	// Basic request counters for observability (no scoring)
	Metrics ProviderMetrics `json:"metrics"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderMetrics tracks basic runtime counters for observability.
// Scoring fields have been removed — TokenHub handles provider quality internally.
type ProviderMetrics struct {
	TotalRequests   int64     `json:"total_requests"`
	SuccessRequests int64     `json:"success_requests"`
	FailedRequests  int64     `json:"failed_requests"`
	LastRequestAt   time.Time `json:"last_request_at"`
	TotalTokens     int64     `json:"total_tokens"`
}

// VersionedEntity interface implementation for Provider
func (p *Provider) GetEntityType() models.EntityType          { return models.EntityTypeProvider }
func (p *Provider) GetSchemaVersion() models.SchemaVersion    { return p.EntityMetadata.SchemaVersion }
func (p *Provider) SetSchemaVersion(v models.SchemaVersion)   { p.EntityMetadata.SchemaVersion = v }
func (p *Provider) GetEntityMetadata() *models.EntityMetadata { return &p.EntityMetadata }
func (p *Provider) GetID() string                             { return p.ID }

// RecordSuccess records a successful provider request.
func (p *Provider) RecordSuccess(latencyMs int64, tokens int64) {
	p.Metrics.TotalRequests++
	p.Metrics.SuccessRequests++
	p.Metrics.LastRequestAt = time.Now()
	if tokens > 0 {
		p.Metrics.TotalTokens += tokens
	}
}

// RecordFailure records a failed provider request.
func (p *Provider) RecordFailure() {
	p.Metrics.TotalRequests++
	p.Metrics.FailedRequests++
	p.Metrics.LastRequestAt = time.Now()
}
