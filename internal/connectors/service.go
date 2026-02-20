package connectors

import (
	"context"

	pkgconnectors "github.com/jordanhubbard/loom/pkg/connectors"
)

// ConnectorService abstracts connector operations so that the API layer can
// use either the local Manager or the remote gRPC client interchangeably.
type ConnectorService interface {
	ListConnectors() []ConnectorInfo
	GetConnector(ctx context.Context, id string) (*ConnectorInfo, error)
	AddConnector(ctx context.Context, cfg pkgconnectors.Config) (string, error)
	UpdateConnector(ctx context.Context, id string, cfg pkgconnectors.Config) error
	RemoveConnector(ctx context.Context, id string) error
	HealthCheck(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, error)
	GetHealthStatus() map[string]pkgconnectors.ConnectorStatus
	TestConnector(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, string, error)
	Close() error
}

// ConnectorInfo is a transport-independent representation of a connector.
type ConnectorInfo struct {
	ID          string
	Name        string
	Type        pkgconnectors.ConnectorType
	Mode        pkgconnectors.ConnectionMode
	Enabled     bool
	Description string
	Endpoint    string
	Status      pkgconnectors.ConnectorStatus
	Tags        []string
	Metadata    map[string]string
}

// LocalService wraps the in-process pkg/connectors.Manager.
type LocalService struct {
	mgr *pkgconnectors.Manager
}

// NewLocalService creates a ConnectorService backed by a local Manager.
func NewLocalService(mgr *pkgconnectors.Manager) *LocalService {
	return &LocalService{mgr: mgr}
}

func (s *LocalService) ListConnectors() []ConnectorInfo {
	all := s.mgr.ListConnectors()
	health := s.mgr.GetHealthStatus()
	infos := make([]ConnectorInfo, 0, len(all))
	for _, c := range all {
		cfg := c.GetConfig()
		st := health[c.ID()]
		if st == "" {
			st = pkgconnectors.ConnectorStatusUnknown
		}
		infos = append(infos, ConnectorInfo{
			ID:          c.ID(),
			Name:        c.Name(),
			Type:        c.Type(),
			Mode:        cfg.Mode,
			Enabled:     cfg.Enabled,
			Description: c.Description(),
			Endpoint:    c.GetEndpoint(),
			Status:      st,
			Tags:        cfg.Tags,
			Metadata:    cfg.Metadata,
		})
	}
	return infos
}

func (s *LocalService) GetConnector(_ context.Context, id string) (*ConnectorInfo, error) {
	c, err := s.mgr.GetConnector(id)
	if err != nil {
		return nil, err
	}
	cfg := c.GetConfig()
	st := s.mgr.GetConnectorHealth(id)
	return &ConnectorInfo{
		ID:          c.ID(),
		Name:        c.Name(),
		Type:        c.Type(),
		Mode:        cfg.Mode,
		Enabled:     cfg.Enabled,
		Description: c.Description(),
		Endpoint:    c.GetEndpoint(),
		Status:      st,
		Tags:        cfg.Tags,
		Metadata:    cfg.Metadata,
	}, nil
}

func (s *LocalService) AddConnector(_ context.Context, cfg pkgconnectors.Config) (string, error) {
	if err := s.mgr.AddConnector(cfg); err != nil {
		return "", err
	}
	_ = s.mgr.SaveConfig()
	return cfg.ID, nil
}

func (s *LocalService) UpdateConnector(_ context.Context, id string, cfg pkgconnectors.Config) error {
	return s.mgr.UpdateConnector(id, cfg)
}

func (s *LocalService) RemoveConnector(_ context.Context, id string) error {
	return s.mgr.RemoveConnector(id)
}

func (s *LocalService) HealthCheck(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, error) {
	c, err := s.mgr.GetConnector(id)
	if err != nil {
		return pkgconnectors.ConnectorStatusUnknown, err
	}
	return c.HealthCheck(ctx)
}

func (s *LocalService) GetHealthStatus() map[string]pkgconnectors.ConnectorStatus {
	return s.mgr.GetHealthStatus()
}

func (s *LocalService) TestConnector(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, string, error) {
	c, err := s.mgr.GetConnector(id)
	if err != nil {
		return pkgconnectors.ConnectorStatusUnknown, "", err
	}
	st, err := c.HealthCheck(ctx)
	return st, c.GetEndpoint(), err
}

func (s *LocalService) Close() error {
	return s.mgr.Close()
}
