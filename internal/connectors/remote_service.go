package connectors

import (
	"context"
	"fmt"

	pb "github.com/jordanhubbard/loom/api/proto/connectors"
	pkgconnectors "github.com/jordanhubbard/loom/pkg/connectors"
)

// RemoteService wraps the gRPC client and implements ConnectorService.
type RemoteService struct {
	client *GRPCClient
}

// NewRemoteService creates a ConnectorService backed by a remote gRPC server.
func NewRemoteService(addr string) (*RemoteService, error) {
	client, err := NewGRPCClient(addr)
	if err != nil {
		return nil, err
	}
	return &RemoteService{client: client}, nil
}

func protoInfoToConnectorInfo(p *pb.ConnectorInfo) ConnectorInfo {
	return ConnectorInfo{
		ID:          p.Id,
		Name:        p.Name,
		Type:        protoTypeToGo(p.Type),
		Mode:        protoModeToGo(p.Mode),
		Enabled:     p.Enabled,
		Description: p.Description,
		Endpoint:    p.Endpoint,
		Status:      protoStatusToGo(p.Status),
		Tags:        p.Tags,
		Metadata:    p.Metadata,
	}
}

func protoStatusToGo(s pb.ConnectorStatus) pkgconnectors.ConnectorStatus {
	switch s {
	case pb.ConnectorStatus_CONNECTOR_STATUS_HEALTHY:
		return pkgconnectors.ConnectorStatusHealthy
	case pb.ConnectorStatus_CONNECTOR_STATUS_UNHEALTHY:
		return pkgconnectors.ConnectorStatusUnhealthy
	case pb.ConnectorStatus_CONNECTOR_STATUS_UNAVAILABLE:
		return pkgconnectors.ConnectorStatusUnavailable
	default:
		return pkgconnectors.ConnectorStatusUnknown
	}
}

func (s *RemoteService) ListConnectors() []ConnectorInfo {
	infos, err := s.client.ListConnectors(context.Background())
	if err != nil {
		return nil
	}
	out := make([]ConnectorInfo, 0, len(infos))
	for _, p := range infos {
		out = append(out, protoInfoToConnectorInfo(p))
	}
	return out
}

func (s *RemoteService) GetConnector(ctx context.Context, id string) (*ConnectorInfo, error) {
	info, _, err := s.client.GetConnector(ctx, id)
	if err != nil {
		return nil, err
	}
	ci := protoInfoToConnectorInfo(info)
	return &ci, nil
}

func (s *RemoteService) AddConnector(ctx context.Context, cfg pkgconnectors.Config) (string, error) {
	pcfg := goConfigToProto(cfg)
	return s.client.RegisterConnector(ctx, pcfg)
}

func (s *RemoteService) UpdateConnector(ctx context.Context, id string, cfg pkgconnectors.Config) error {
	if err := s.client.RemoveConnector(ctx, id); err != nil {
		return fmt.Errorf("failed to remove old connector during update: %w", err)
	}
	cfg.ID = id
	pcfg := goConfigToProto(cfg)
	_, err := s.client.RegisterConnector(ctx, pcfg)
	return err
}

func (s *RemoteService) RemoveConnector(ctx context.Context, id string) error {
	return s.client.RemoveConnector(ctx, id)
}

func (s *RemoteService) HealthCheck(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, error) {
	resp, err := s.client.HealthCheck(ctx, id)
	if err != nil {
		return pkgconnectors.ConnectorStatusUnknown, err
	}
	return protoStatusToGo(resp.Status), nil
}

func (s *RemoteService) GetHealthStatus() map[string]pkgconnectors.ConnectorStatus {
	statuses, err := s.client.HealthCheckAll(context.Background())
	if err != nil {
		return nil
	}
	out := make(map[string]pkgconnectors.ConnectorStatus, len(statuses))
	for id, st := range statuses {
		out[id] = protoStatusToGo(st)
	}
	return out
}

func (s *RemoteService) TestConnector(ctx context.Context, id string) (pkgconnectors.ConnectorStatus, string, error) {
	info, _, err := s.client.GetConnector(ctx, id)
	if err != nil {
		return pkgconnectors.ConnectorStatusUnknown, "", err
	}
	resp, err := s.client.HealthCheck(ctx, id)
	if err != nil {
		return pkgconnectors.ConnectorStatusUnknown, info.Endpoint, err
	}
	return protoStatusToGo(resp.Status), info.Endpoint, nil
}

func (s *RemoteService) Close() error {
	return s.client.Close()
}

// goConfigToProto converts a Go config to the protobuf config format.
func goConfigToProto(cfg pkgconnectors.Config) *pb.ConnectorConfig {
	pcfg := &pb.ConnectorConfig{
		Id:          cfg.ID,
		Name:        cfg.Name,
		Type:        goTypeToProto(cfg.Type),
		Mode:        goModeToProto(cfg.Mode),
		Enabled:     cfg.Enabled,
		Description: cfg.Description,
		Host:        cfg.Host,
		Port:        int32(cfg.Port),
		Scheme:      cfg.Scheme,
		BasePath:    cfg.BasePath,
		Metadata:    cfg.Metadata,
		Tags:        cfg.Tags,
	}
	if cfg.Auth != nil {
		pcfg.Auth = &pb.AuthConfig{
			Type:     cfg.Auth.Type,
			Username: cfg.Auth.Username,
			Password: cfg.Auth.Password,
			Token:    cfg.Auth.Token,
			ApiKey:   cfg.Auth.APIKey,
			Headers:  cfg.Auth.Headers,
		}
	}
	if cfg.HealthCheck != nil {
		pcfg.HealthCheck = &pb.HealthCheckConfig{
			Enabled:    cfg.HealthCheck.Enabled,
			IntervalMs: cfg.HealthCheck.Interval.Milliseconds(),
			TimeoutMs:  cfg.HealthCheck.Timeout.Milliseconds(),
			Path:       cfg.HealthCheck.Path,
		}
	}
	return pcfg
}
