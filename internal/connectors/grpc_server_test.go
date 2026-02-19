package connectors

import (
	"context"
	"testing"
	"time"

	pb "github.com/jordanhubbard/loom/api/proto/connectors"
	pkgconnectors "github.com/jordanhubbard/loom/pkg/connectors"
)

func newTestManager(t *testing.T) *pkgconnectors.Manager {
	t.Helper()
	dir := t.TempDir()
	mgr := pkgconnectors.NewManager(dir + "/connectors.yaml")
	// Load will create default config; ignore errors in tests
	_ = mgr.LoadConfig()
	return mgr
}

func TestListConnectors_ReturnsDefaults(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	resp, err := srv.ListConnectors(context.Background(), &pb.ListConnectorsRequest{})
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	if len(resp.Connectors) == 0 {
		t.Fatal("expected at least one connector in default config")
	}
}

func TestGetConnector_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	_, err := srv.GetConnector(context.Background(), &pb.GetConnectorRequest{Id: "does-not-exist"})
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestGetConnector_Found(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	// Default config includes prometheus
	resp, err := srv.GetConnector(context.Background(), &pb.GetConnectorRequest{Id: "prometheus"})
	if err != nil {
		t.Fatalf("GetConnector: %v", err)
	}
	if resp.Connector.Id != "prometheus" {
		t.Fatalf("expected prometheus, got %s", resp.Connector.Id)
	}
	if resp.Config == nil {
		t.Fatal("expected config in response")
	}
}

func TestRegisterAndRemoveConnector(t *testing.T) {
	// Use an empty manager (no LoadConfig) to avoid pre-registered connectors.
	dir := t.TempDir()
	mgr := pkgconnectors.NewManager(dir + "/connectors.yaml")
	srv := NewGRPCServer(mgr)

	// prometheus is a known connector type supported by the factory.
	cfg := &pb.ConnectorConfig{
		Id:      "prometheus",
		Name:    "Prometheus",
		Type:    pb.ConnectorType_CONNECTOR_TYPE_OBSERVABILITY,
		Mode:    pb.ConnectionMode_CONNECTION_MODE_LOCAL,
		Enabled: true,
		Host:    "localhost",
		Port:    9090,
		Scheme:  "http",
		HealthCheck: &pb.HealthCheckConfig{
			Enabled:    true,
			IntervalMs: 30000,
			TimeoutMs:  5000,
			Path:       "/-/healthy",
		},
	}

	// Register
	regResp, err := srv.RegisterConnector(context.Background(), &pb.RegisterConnectorRequest{Config: cfg})
	if err != nil {
		t.Fatalf("RegisterConnector: %v", err)
	}
	if !regResp.Success {
		t.Fatalf("expected success, got message: %s", regResp.Message)
	}

	// List and verify
	listResp, _ := srv.ListConnectors(context.Background(), &pb.ListConnectorsRequest{})
	found := false
	for _, c := range listResp.Connectors {
		if c.Id == "prometheus" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("newly registered connector not found in list")
	}

	// Remove
	rmResp, err := srv.RemoveConnector(context.Background(), &pb.RemoveConnectorRequest{Id: "prometheus"})
	if err != nil {
		t.Fatalf("RemoveConnector: %v", err)
	}
	if !rmResp.Success {
		t.Fatalf("expected success removing connector")
	}
}

func TestHealthCheck_ConnectorNotFound(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	_, err := srv.HealthCheck(context.Background(), &pb.HealthCheckRequest{Id: "missing"})
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestHealthCheckAll_ReturnsStatuses(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := srv.HealthCheckAll(ctx, &pb.HealthCheckAllRequest{})
	if err != nil {
		t.Fatalf("HealthCheckAll: %v", err)
	}
	if len(resp.Statuses) == 0 {
		t.Fatal("expected at least one status")
	}
}

func TestListConnectors_FilterByType(t *testing.T) {
	mgr := newTestManager(t)
	srv := NewGRPCServer(mgr)

	obsType := pb.ConnectorType_CONNECTOR_TYPE_OBSERVABILITY
	resp, err := srv.ListConnectors(context.Background(), &pb.ListConnectorsRequest{
		Type: &obsType,
	})
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	for _, c := range resp.Connectors {
		if c.Type != pb.ConnectorType_CONNECTOR_TYPE_OBSERVABILITY {
			t.Errorf("expected observability type, got %v", c.Type)
		}
	}
}
