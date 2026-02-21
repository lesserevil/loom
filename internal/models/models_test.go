package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// TestWorkStatus tests WorkStatus constants
func TestWorkStatus(t *testing.T) {
	tests := []struct {
		name   string
		status WorkStatus
		want   string
	}{
		{"Pending", WorkStatusPending, "pending"},
		{"InProgress", WorkStatusInProgress, "in_progress"},
		{"Completed", WorkStatusCompleted, "completed"},
		{"Failed", WorkStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("WorkStatus = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

// TestWork tests Work struct
func TestWork(t *testing.T) {
	now := time.Now()
	work := Work{
		ID:          "work-1",
		Description: "Test work",
		Status:      WorkStatusPending,
		AssignedTo:  "agent-1",
		CreatedAt:   now,
		UpdatedAt:   now,
		Result:      "success",
	}

	if work.ID != "work-1" {
		t.Errorf("Work.ID = %q, want %q", work.ID, "work-1")
	}

	if work.Status != WorkStatusPending {
		t.Errorf("Work.Status = %q, want %q", work.Status, WorkStatusPending)
	}

	// Test JSON marshaling
	data, err := json.Marshal(work)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled Work
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.ID != work.ID {
		t.Errorf("Unmarshaled ID = %q, want %q", unmarshaled.ID, work.ID)
	}

	if unmarshaled.Description != work.Description {
		t.Errorf("Unmarshaled Description = %q, want %q", unmarshaled.Description, work.Description)
	}
}

// TestWorkOmitEmpty tests omitempty JSON tags
func TestWorkOmitEmpty(t *testing.T) {
	work := Work{
		ID:          "work-1",
		Description: "Test",
		Status:      WorkStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		// AssignedTo and Result are empty
	}

	data, err := json.Marshal(work)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// AssignedTo and Result should be omitted
	if _, exists := result["assigned_to"]; exists {
		t.Error("Expected assigned_to to be omitted when empty")
	}

	if _, exists := result["result"]; exists {
		t.Error("Expected result to be omitted when empty")
	}
}

// TestAgentCommunication tests AgentCommunication struct
func TestAgentCommunication(t *testing.T) {
	now := time.Now()
	comm := AgentCommunication{
		ID:        "comm-1",
		FromAgent: "agent-1",
		ToAgent:   "agent-2",
		Message:   "Hello",
		Timestamp: now,
	}

	if comm.FromAgent != "agent-1" {
		t.Errorf("FromAgent = %q, want %q", comm.FromAgent, "agent-1")
	}

	if comm.ToAgent != "agent-2" {
		t.Errorf("ToAgent = %q, want %q", comm.ToAgent, "agent-2")
	}

	// Test JSON marshaling
	data, err := json.Marshal(comm)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled AgentCommunication
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Message != comm.Message {
		t.Errorf("Unmarshaled Message = %q, want %q", unmarshaled.Message, comm.Message)
	}
}

// TestCostType tests CostType constants
func TestCostType(t *testing.T) {
	tests := []struct {
		name     string
		costType CostType
		want     string
	}{
		{"Fixed", CostTypeFixed, "fixed"},
		{"Variable", CostTypeVariable, "variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.costType) != tt.want {
				t.Errorf("CostType = %q, want %q", tt.costType, tt.want)
			}
		})
	}
}

// TestServiceEndpoint tests ServiceEndpoint struct
func TestServiceEndpoint(t *testing.T) {
	now := time.Now()
	endpoint := ServiceEndpoint{
		ID:           "svc-1",
		Name:         "OpenAI GPT-4",
		URL:          "https://api.openai.com",
		Type:         "openai",
		IsActive:     true,
		CostType:     CostTypeVariable,
		CostPerToken: 0.03,
		TokensUsed:   1000000,
		TotalCost:    30.0,
		RequestCount: 500,
		LastActive:   now,
		CreatedAt:    now,
	}

	if endpoint.CostType != CostTypeVariable {
		t.Errorf("CostType = %q, want %q", endpoint.CostType, CostTypeVariable)
	}

	if endpoint.CostPerToken != 0.03 {
		t.Errorf("CostPerToken = %f, want %f", endpoint.CostPerToken, 0.03)
	}

	if endpoint.TokensUsed != 1000000 {
		t.Errorf("TokensUsed = %d, want %d", endpoint.TokensUsed, 1000000)
	}

	// Test JSON marshaling
	data, err := json.Marshal(endpoint)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled ServiceEndpoint
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Name != endpoint.Name {
		t.Errorf("Unmarshaled Name = %q, want %q", unmarshaled.Name, endpoint.Name)
	}
}

// TestServiceEndpointFixedCost tests fixed cost service
func TestServiceEndpointFixedCost(t *testing.T) {
	endpoint := ServiceEndpoint{
		ID:        "svc-2",
		Name:      "Local Ollama",
		Type:      "ollama",
		CostType:  CostTypeFixed,
		FixedCost: 100.0,
		IsActive:  true,
	}

	if endpoint.CostType != CostTypeFixed {
		t.Errorf("CostType = %q, want %q", endpoint.CostType, CostTypeFixed)
	}

	if endpoint.FixedCost != 100.0 {
		t.Errorf("FixedCost = %f, want %f", endpoint.FixedCost, 100.0)
	}
}

// TestTraffic tests Traffic struct
func TestTraffic(t *testing.T) {
	now := time.Now()
	traffic := Traffic{
		ServiceID:     "svc-1",
		BytesSent:     1024,
		BytesReceived: 2048,
		RequestCount:  10,
		Timestamp:     now,
	}

	if traffic.BytesSent != 1024 {
		t.Errorf("BytesSent = %d, want %d", traffic.BytesSent, 1024)
	}

	if traffic.BytesReceived != 2048 {
		t.Errorf("BytesReceived = %d, want %d", traffic.BytesReceived, 2048)
	}

	// Test JSON marshaling
	data, err := json.Marshal(traffic)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled Traffic
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.RequestCount != traffic.RequestCount {
		t.Errorf("Unmarshaled RequestCount = %d, want %d", unmarshaled.RequestCount, traffic.RequestCount)
	}
}

// TestCreateWorkRequest tests CreateWorkRequest struct
func TestCreateWorkRequest(t *testing.T) {
	req := CreateWorkRequest{
		Description: "New work item",
	}

	if req.Description != "New work item" {
		t.Errorf("Description = %q, want %q", req.Description, "New work item")
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled CreateWorkRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Description != req.Description {
		t.Errorf("Unmarshaled Description = %q, want %q", unmarshaled.Description, req.Description)
	}
}

// TestUpdateServiceCostRequest tests UpdateServiceCostRequest struct
func TestUpdateServiceCostRequest(t *testing.T) {
	costPerToken := 0.05
	fixedCost := 200.0

	req := UpdateServiceCostRequest{
		CostType:     CostTypeVariable,
		CostPerToken: &costPerToken,
		FixedCost:    &fixedCost,
	}

	if req.CostType != CostTypeVariable {
		t.Errorf("CostType = %q, want %q", req.CostType, CostTypeVariable)
	}

	if req.CostPerToken == nil || *req.CostPerToken != 0.05 {
		t.Errorf("CostPerToken = %v, want %f", req.CostPerToken, 0.05)
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled UpdateServiceCostRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.CostPerToken == nil || *unmarshaled.CostPerToken != *req.CostPerToken {
		t.Errorf("Unmarshaled CostPerToken = %v, want %v", unmarshaled.CostPerToken, req.CostPerToken)
	}
}

// TestUpdateServiceCostRequestOmitEmpty tests omitempty on pointers
func TestUpdateServiceCostRequestOmitEmpty(t *testing.T) {
	req := UpdateServiceCostRequest{
		CostType: CostTypeFixed,
		// No CostPerToken or FixedCost
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := result["cost_per_token"]; exists {
		t.Error("Expected cost_per_token to be omitted when nil")
	}

	if _, exists := result["fixed_cost"]; exists {
		t.Error("Expected fixed_cost to be omitted when nil")
	}
}

// TestProvider tests Provider struct initialization
func TestProvider(t *testing.T) {
	now := time.Now()
	provider := Provider{
		ID:              "prov-1",
		Name:            "OpenAI",
		Type:            "openai",
		Endpoint:        "https://api.openai.com",
		Model:           "gpt-4",
		ConfiguredModel: "gpt-4-turbo",
		Status:          "active",
		RequiresKey:     true,
		KeyID:           "key-1",
		IsShared:        false,
		ContextWindow:   128000,
		Tags:            []string{"production", "fast"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if provider.Name != "OpenAI" {
		t.Errorf("Provider.Name = %q, want %q", provider.Name, "OpenAI")
	}

	if provider.ContextWindow != 128000 {
		t.Errorf("Provider.ContextWindow = %d, want %d", provider.ContextWindow, 128000)
	}
}

// TestProviderGetEntityType tests Provider interface methods
func TestProviderGetEntityType(t *testing.T) {
	provider := &Provider{ID: "prov-1"}

	if provider.GetEntityType() != models.EntityTypeProvider {
		t.Errorf("GetEntityType() = %v, want %v", provider.GetEntityType(), models.EntityTypeProvider)
	}

	if provider.GetID() != "prov-1" {
		t.Errorf("GetID() = %q, want %q", provider.GetID(), "prov-1")
	}
}

// TestProviderSchemaVersion tests schema version methods
func TestProviderSchemaVersion(t *testing.T) {
	provider := &Provider{}

	version := models.SchemaVersion("v2")
	provider.SetSchemaVersion(version)

	if provider.GetSchemaVersion() != version {
		t.Errorf("GetSchemaVersion() = %v, want %v", provider.GetSchemaVersion(), version)
	}

	metadata := provider.GetEntityMetadata()
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if metadata.SchemaVersion != version {
		t.Errorf("metadata.SchemaVersion = %v, want %v", metadata.SchemaVersion, version)
	}
}

func TestProviderRecordSuccess(t *testing.T) {
	provider := &Provider{
		ID:     "prov-1",
		Status: "active",
	}

	provider.RecordSuccess(1000, 500)

	if provider.Metrics.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want %d", provider.Metrics.TotalRequests, 1)
	}
	if provider.Metrics.SuccessRequests != 1 {
		t.Errorf("SuccessRequests = %d, want %d", provider.Metrics.SuccessRequests, 1)
	}
	if provider.Metrics.TotalTokens != 500 {
		t.Errorf("TotalTokens = %d, want %d", provider.Metrics.TotalTokens, 500)
	}

	provider.RecordSuccess(2000, 300)

	if provider.Metrics.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want %d", provider.Metrics.TotalRequests, 2)
	}
	if provider.Metrics.TotalTokens != 800 {
		t.Errorf("TotalTokens = %d, want %d", provider.Metrics.TotalTokens, 800)
	}
}

func TestProviderRecordFailure(t *testing.T) {
	provider := &Provider{
		ID:     "prov-1",
		Status: "active",
	}

	provider.RecordSuccess(1000, 100)
	provider.RecordFailure()

	if provider.Metrics.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want %d", provider.Metrics.TotalRequests, 2)
	}
	if provider.Metrics.SuccessRequests != 1 {
		t.Errorf("SuccessRequests = %d, want %d", provider.Metrics.SuccessRequests, 1)
	}
	if provider.Metrics.FailedRequests != 1 {
		t.Errorf("FailedRequests = %d, want %d", provider.Metrics.FailedRequests, 1)
	}
}

func TestProviderRecordSuccessWithZeroTokens(t *testing.T) {
	provider := &Provider{
		ID:     "prov-1",
		Status: "active",
	}

	provider.RecordSuccess(1000, 0)

	if provider.Metrics.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want %d", provider.Metrics.TotalRequests, 1)
	}
	if provider.Metrics.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want %d", provider.Metrics.TotalTokens, 0)
	}
}
