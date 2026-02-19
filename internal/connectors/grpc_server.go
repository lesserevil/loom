// Package connectors provides the gRPC ConnectorsService implementation.
package connectors

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/jordanhubbard/loom/api/proto/connectors"
	pkgconnectors "github.com/jordanhubbard/loom/pkg/connectors"
)

// GRPCServer implements pb.ConnectorsServiceServer by delegating to a
// pkg/connectors.Manager instance.
type GRPCServer struct {
	pb.UnimplementedConnectorsServiceServer
	mgr *pkgconnectors.Manager
}

// NewGRPCServer creates a new ConnectorsService gRPC server.
func NewGRPCServer(mgr *pkgconnectors.Manager) *GRPCServer {
	return &GRPCServer{mgr: mgr}
}

// ListConnectors returns all registered connectors, optionally filtered.
func (s *GRPCServer) ListConnectors(_ context.Context, req *pb.ListConnectorsRequest) (*pb.ListConnectorsResponse, error) {
	var connectors []pkgconnectors.Connector

	if req.Type != nil {
		goType := protoTypeToGo(req.GetType())
		connectors = s.mgr.ListConnectorsByType(goType)
	} else {
		connectors = s.mgr.ListConnectors()
	}

	health := s.mgr.GetHealthStatus()

	infos := make([]*pb.ConnectorInfo, 0, len(connectors))
	for _, c := range connectors {
		cfg := c.GetConfig()
		if req.EnabledOnly != nil && req.GetEnabledOnly() && !cfg.Enabled {
			continue
		}
		infos = append(infos, connectorToProtoInfo(c, health[c.ID()]))
	}
	return &pb.ListConnectorsResponse{Connectors: infos}, nil
}

// GetConnector retrieves a specific connector by ID.
func (s *GRPCServer) GetConnector(_ context.Context, req *pb.GetConnectorRequest) (*pb.GetConnectorResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	c, err := s.mgr.GetConnector(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "connector %s not found", req.Id)
	}

	health := s.mgr.GetConnectorHealth(req.Id)
	return &pb.GetConnectorResponse{
		Connector: connectorToProtoInfo(c, health),
		Config:    connectorToProtoConfig(c),
	}, nil
}

// HealthCheck checks the health of a specific connector.
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	c, err := s.mgr.GetConnector(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "connector %s not found", req.Id)
	}

	start := time.Now()
	st, _ := c.HealthCheck(ctx)
	elapsed := time.Since(start).Milliseconds()

	return &pb.HealthCheckResponse{
		Id:             req.Id,
		Status:         goStatusToProto(st),
		ResponseTimeMs: elapsed,
	}, nil
}

// HealthCheckAll checks health of all connectors.
func (s *GRPCServer) HealthCheckAll(ctx context.Context, req *pb.HealthCheckAllRequest) (*pb.HealthCheckAllResponse, error) {
	var connectors []pkgconnectors.Connector
	if req.Type != nil {
		connectors = s.mgr.ListConnectorsByType(protoTypeToGo(req.GetType()))
	} else {
		connectors = s.mgr.ListConnectors()
	}

	statuses := make(map[string]pb.ConnectorStatus, len(connectors))
	for _, c := range connectors {
		st, _ := c.HealthCheck(ctx)
		statuses[c.ID()] = goStatusToProto(st)
	}
	return &pb.HealthCheckAllResponse{Statuses: statuses}, nil
}

// RegisterConnector adds a new connector.
func (s *GRPCServer) RegisterConnector(_ context.Context, req *pb.RegisterConnectorRequest) (*pb.RegisterConnectorResponse, error) {
	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}

	cfg := protoConfigToGo(req.Config)
	if err := s.mgr.AddConnector(cfg); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register connector: %v", err)
	}

	return &pb.RegisterConnectorResponse{
		Success:     true,
		Message:     fmt.Sprintf("connector %s registered", cfg.ID),
		ConnectorId: cfg.ID,
	}, nil
}

// RemoveConnector removes a connector.
func (s *GRPCServer) RemoveConnector(_ context.Context, req *pb.RemoveConnectorRequest) (*pb.RemoveConnectorResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.mgr.RemoveConnector(req.Id); err != nil {
		return nil, status.Errorf(codes.NotFound, "connector %s not found", req.Id)
	}

	return &pb.RemoveConnectorResponse{
		Success: true,
		Message: fmt.Sprintf("connector %s removed", req.Id),
	}, nil
}

// --- helpers ---

func connectorToProtoInfo(c pkgconnectors.Connector, health pkgconnectors.ConnectorStatus) *pb.ConnectorInfo {
	cfg := c.GetConfig()
	info := &pb.ConnectorInfo{
		Id:          c.ID(),
		Name:        c.Name(),
		Type:        goTypeToProto(c.Type()),
		Description: c.Description(),
		Endpoint:    c.GetEndpoint(),
		Enabled:     cfg.Enabled,
		Status:      goStatusToProto(health),
		Mode:        goModeToProto(cfg.Mode),
		Metadata:    cfg.Metadata,
		Tags:        cfg.Tags,
	}
	return info
}

func connectorToProtoConfig(c pkgconnectors.Connector) *pb.ConnectorConfig {
	cfg := c.GetConfig()
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

func protoConfigToGo(p *pb.ConnectorConfig) pkgconnectors.Config {
	cfg := pkgconnectors.Config{
		ID:          p.Id,
		Name:        p.Name,
		Type:        protoTypeToGo(p.Type),
		Mode:        protoModeToGo(p.Mode),
		Enabled:     p.Enabled,
		Description: p.Description,
		Host:        p.Host,
		Port:        int(p.Port),
		Scheme:      p.Scheme,
		BasePath:    p.BasePath,
		Metadata:    p.Metadata,
		Tags:        p.Tags,
	}
	if p.Auth != nil {
		cfg.Auth = &pkgconnectors.AuthConfig{
			Type:     p.Auth.Type,
			Username: p.Auth.Username,
			Password: p.Auth.Password,
			Token:    p.Auth.Token,
			APIKey:   p.Auth.ApiKey,
			Headers:  p.Auth.Headers,
		}
	}
	if p.HealthCheck != nil {
		cfg.HealthCheck = &pkgconnectors.HealthCheckConfig{
			Enabled:  p.HealthCheck.Enabled,
			Interval: time.Duration(p.HealthCheck.IntervalMs) * time.Millisecond,
			Timeout:  time.Duration(p.HealthCheck.TimeoutMs) * time.Millisecond,
			Path:     p.HealthCheck.Path,
		}
	}
	return cfg
}

func goTypeToProto(t pkgconnectors.ConnectorType) pb.ConnectorType {
	switch t {
	case pkgconnectors.ConnectorTypeObservability:
		return pb.ConnectorType_CONNECTOR_TYPE_OBSERVABILITY
	case pkgconnectors.ConnectorTypeAgent:
		return pb.ConnectorType_CONNECTOR_TYPE_AGENT
	case pkgconnectors.ConnectorTypeStorage:
		return pb.ConnectorType_CONNECTOR_TYPE_STORAGE
	case pkgconnectors.ConnectorTypeMessaging:
		return pb.ConnectorType_CONNECTOR_TYPE_MESSAGING
	case pkgconnectors.ConnectorTypeDatabase:
		return pb.ConnectorType_CONNECTOR_TYPE_DATABASE
	case pkgconnectors.ConnectorTypeCustom:
		return pb.ConnectorType_CONNECTOR_TYPE_CUSTOM
	default:
		return pb.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED
	}
}

func protoTypeToGo(t pb.ConnectorType) pkgconnectors.ConnectorType {
	switch t {
	case pb.ConnectorType_CONNECTOR_TYPE_OBSERVABILITY:
		return pkgconnectors.ConnectorTypeObservability
	case pb.ConnectorType_CONNECTOR_TYPE_AGENT:
		return pkgconnectors.ConnectorTypeAgent
	case pb.ConnectorType_CONNECTOR_TYPE_STORAGE:
		return pkgconnectors.ConnectorTypeStorage
	case pb.ConnectorType_CONNECTOR_TYPE_MESSAGING:
		return pkgconnectors.ConnectorTypeMessaging
	case pb.ConnectorType_CONNECTOR_TYPE_DATABASE:
		return pkgconnectors.ConnectorTypeDatabase
	case pb.ConnectorType_CONNECTOR_TYPE_CUSTOM:
		return pkgconnectors.ConnectorTypeCustom
	default:
		return pkgconnectors.ConnectorTypeCustom
	}
}

func goModeToProto(m pkgconnectors.ConnectionMode) pb.ConnectionMode {
	switch m {
	case pkgconnectors.ConnectionModeLocal:
		return pb.ConnectionMode_CONNECTION_MODE_LOCAL
	case pkgconnectors.ConnectionModeRemote:
		return pb.ConnectionMode_CONNECTION_MODE_REMOTE
	default:
		return pb.ConnectionMode_CONNECTION_MODE_UNSPECIFIED
	}
}

func protoModeToGo(m pb.ConnectionMode) pkgconnectors.ConnectionMode {
	switch m {
	case pb.ConnectionMode_CONNECTION_MODE_LOCAL:
		return pkgconnectors.ConnectionModeLocal
	case pb.ConnectionMode_CONNECTION_MODE_REMOTE:
		return pkgconnectors.ConnectionModeRemote
	default:
		return pkgconnectors.ConnectionModeLocal
	}
}

func goStatusToProto(s pkgconnectors.ConnectorStatus) pb.ConnectorStatus {
	switch s {
	case pkgconnectors.ConnectorStatusHealthy:
		return pb.ConnectorStatus_CONNECTOR_STATUS_HEALTHY
	case pkgconnectors.ConnectorStatusUnhealthy:
		return pb.ConnectorStatus_CONNECTOR_STATUS_UNHEALTHY
	case pkgconnectors.ConnectorStatusUnavailable:
		return pb.ConnectorStatus_CONNECTOR_STATUS_UNAVAILABLE
	default:
		return pb.ConnectorStatus_CONNECTOR_STATUS_UNKNOWN
	}
}
