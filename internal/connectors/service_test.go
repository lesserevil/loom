package connectors

import (
	"context"
	"testing"

	pkgconnectors "github.com/jordanhubbard/loom/pkg/connectors"
)

func TestLocalService_ListConnectors(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	list := svc.ListConnectors()
	if len(list) == 0 {
		t.Fatal("expected at least one connector in default config")
	}
	var found bool
	for _, c := range list {
		if c.ID == "prometheus" {
			found = true
			if c.Type != pkgconnectors.ConnectorTypeObservability {
				t.Errorf("expected observability type for prometheus, got %s", c.Type)
			}
		}
	}
	if !found {
		t.Error("prometheus connector not found in list")
	}
}

func TestLocalService_GetConnector(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	c, err := svc.GetConnector(context.Background(), "prometheus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != "prometheus" {
		t.Errorf("expected id=prometheus, got %s", c.ID)
	}
}

func TestLocalService_GetConnector_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	_, err := svc.GetConnector(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestLocalService_GetHealthStatus(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	status := svc.GetHealthStatus()
	if status == nil {
		t.Fatal("expected non-nil health status map")
	}
}

func TestLocalService_HealthCheck_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	_, err := svc.HealthCheck(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestLocalService_RemoveConnector_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	svc := NewLocalService(mgr)

	err := svc.RemoveConnector(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error removing nonexistent connector")
	}
}

func TestConnectorInfo_Fields(t *testing.T) {
	ci := ConnectorInfo{
		ID:       "test",
		Name:     "Test",
		Type:     pkgconnectors.ConnectorTypeCustom,
		Mode:     pkgconnectors.ConnectionModeLocal,
		Enabled:  true,
		Endpoint: "http://localhost:9999",
	}
	if ci.ID != "test" || ci.Name != "Test" {
		t.Error("ConnectorInfo fields not set correctly")
	}
}
