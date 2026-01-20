package models

import (
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// Provider represents an AI engine running on-prem or in the cloud
// Providers may require credentials (keys) to communicate
type Provider struct {
	models.EntityMetadata `json:",inline"`

	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Type                   string    `json:"type"`     // openai, anthropic, local, etc.
	Endpoint               string    `json:"endpoint"` // URL or path to the provider
	Model                  string    `json:"model"`    // Legacy/default model for this provider
	ConfiguredModel        string    `json:"configured_model"`
	SelectedModel          string    `json:"selected_model"`
	SelectionReason        string    `json:"selection_reason"`
	ModelScore             float64   `json:"model_score"`
	SelectedGPU            string    `json:"selected_gpu"`
	Description            string    `json:"description"`
	RequiresKey            bool      `json:"requires_key"` // Whether this provider needs API credentials
	KeyID                  string    `json:"key_id"`       // Reference to encrypted key in key manager
	Status                 string    `json:"status"`       // active, inactive, etc.
	LastHeartbeatAt        time.Time `json:"last_heartbeat_at"`
	LastHeartbeatLatencyMs int64     `json:"last_heartbeat_latency_ms"`
	LastHeartbeatError     string    `json:"last_heartbeat_error"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// VersionedEntity interface implementation for Provider
func (p *Provider) GetEntityType() models.EntityType          { return models.EntityTypeProvider }
func (p *Provider) GetSchemaVersion() models.SchemaVersion    { return p.EntityMetadata.SchemaVersion }
func (p *Provider) SetSchemaVersion(v models.SchemaVersion)   { p.EntityMetadata.SchemaVersion = v }
func (p *Provider) GetEntityMetadata() *models.EntityMetadata { return &p.EntityMetadata }
func (p *Provider) GetID() string                             { return p.ID }
