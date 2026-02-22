// Package grpc provides the gRPC server implementation for the Knowledge service.
// It manages detections, actions, databases, system configurations, and system status.
package grpc

import (
	"context"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

// KnowledgeServer implements the KnowledgeService gRPC interface.
type KnowledgeServer struct {
	pb.UnimplementedKnowledgeServiceServer
	redisClient *redis.Client
	startTime   time.Time
}

// NewKnowledgeServer creates a new KnowledgeServer instance.
func NewKnowledgeServer(redisClient *redis.Client) *KnowledgeServer {
	return &KnowledgeServer{
		redisClient: redisClient,
		startTime:   time.Now(),
	}
}

// ===== [DETECTION OPERATIONS] =====

// RegisterDetection registers a new detection in the knowledge base.
func (s *KnowledgeServer) RegisterDetection(ctx context.Context, req *pb.RegisterDetectionRequest) (*pb.DetectionResponse, error) {
	detection := &models.Detection{
		ID:         req.Id,
		Key:        req.Key,
		State:      models.StateActive,
		Severity:   req.Severity,
		Category:   req.Category,
		DatabaseID: req.DatabaseId,
		Value:      req.Value,
		CreatedAt:  time.Unix(req.CreatedAt, 0),
		LastSeen:   time.Now(),
		TTL:        0,
	}

	if err := s.redisClient.RegisterDetection(ctx, detection); err != nil {
		log.Printf("Failed to register detection: %v", err)
		return &pb.DetectionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Detection registered: %s (key: %s)", detection.ID, detection.Key)

	return &pb.DetectionResponse{
		Success:     true,
		Message:     "Detection registered successfully",
		DetectionId: detection.ID,
	}, nil
}

// IsDetectionActive checks if a detection with the given key is active.
func (s *KnowledgeServer) IsDetectionActive(ctx context.Context, req *pb.DetectionKeyRequest) (*pb.DetectionStatusResponse, error) {
	isActive, err := s.redisClient.IsDetectionActive(ctx, req.Key)
	if err != nil {
		log.Printf("Failed to check detection status: %v", err)
		return &pb.DetectionStatusResponse{
			IsActive: false,
		}, nil
	}

	detectionID := ""
	if isActive {
		detectionID, _ = s.redisClient.GetDetectionIDByKey(ctx, req.Key)
	}

	return &pb.DetectionStatusResponse{
		IsActive:    isActive,
		DetectionId: detectionID,
	}, nil
}

// GetActiveDetections returns all active detections for a database.
func (s *KnowledgeServer) GetActiveDetections(ctx context.Context, req *pb.DatabaseFilterRequest) (*pb.DetectionListResponse, error) {
	detections, err := s.redisClient.GetActiveDetections(ctx, req.DatabaseId)
	if err != nil {
		log.Printf("Failed to get active detections: %v", err)
		return &pb.DetectionListResponse{
			Detections: []*pb.Detection{},
		}, nil
	}

	pbDetections := make([]*pb.Detection, 0, len(detections))
	for _, d := range detections {
		pbDetections = append(pbDetections, &pb.Detection{
			Id:         d.ID,
			Key:        d.Key,
			State:      string(d.State),
			Severity:   d.Severity,
			Category:   d.Category,
			DatabaseId: d.DatabaseID,
			Value:      d.Value,
			ActionId:   d.ActionID,
			ResolvedBy: d.ResolvedBy,
			CreatedAt:  d.CreatedAt.Unix(),
			LastSeen:   d.LastSeen.Unix(),
		})
	}

	return &pb.DetectionListResponse{
		Detections: pbDetections,
	}, nil
}

// MarkDetectionResolved marks a detection as resolved.
func (s *KnowledgeServer) MarkDetectionResolved(ctx context.Context, req *pb.ResolveDetectionRequest) (*pb.Response, error) {
	if err := s.redisClient.MarkDetectionResolved(ctx, req.DetectionId, req.Solution); err != nil {
		log.Printf("Failed to mark detection resolved: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Detection resolved: %s (solution: %s)", req.DetectionId, req.Solution)

	return &pb.Response{
		Success: true,
		Message: "Detection marked as resolved",
	}, nil
}

// ===== [ACTION OPERATIONS] =====

// RegisterAction registers a new action in the knowledge base.
func (s *KnowledgeServer) RegisterAction(ctx context.Context, req *pb.RegisterActionRequest) (*pb.ActionResponse, error) {
	action := &models.Action{
		ID:          req.Id,
		DetectionID: req.DetectionId,
		ActionType:  req.ActionType,
		DatabaseID:  req.DatabaseId,
		Status:      models.StatusQueued,
		Message:     "Action queued",
		CreatedAt:   time.Unix(req.CreatedAt, 0),
	}

	if err := s.redisClient.RegisterAction(ctx, action); err != nil {
		log.Printf("Failed to register action: %v", err)
		return &pb.ActionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Action registered: %s (type: %s, detection: %s)", action.ID, action.ActionType, action.DetectionID)

	return &pb.ActionResponse{
		Success:  true,
		Message:  "Action successfully registered",
		ActionId: action.ID,
	}, nil
}

// UpdateActionStatus updates the status of an existing action.
func (s *KnowledgeServer) UpdateActionStatus(ctx context.Context, req *pb.UpdateActionRequest) (*pb.Response, error) {
	if err := s.redisClient.UpdateActionStatus(ctx, req.ActionId, models.ActionStatus(req.Status), req.Message, req.Error); err != nil {
		log.Printf("Failed to update action status: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Action status updated %s => %s", req.ActionId, req.Status)

	return &pb.Response{
		Success: true,
		Message: "Action status updated successfully",
	}, nil
}

// GetPendingActions retrieves all pending actions, optionally filtered by database.
func (s *KnowledgeServer) GetPendingActions(ctx context.Context, req *pb.DatabaseFilterRequest) (*pb.ActionListResponse, error) {
	actions, err := s.redisClient.GetPendingActions(ctx, req.DatabaseId)
	if err != nil {
		log.Printf("Failed to get pending actions: %v", err)
		return &pb.ActionListResponse{
			Actions: []*pb.Action{},
		}, nil
	}

	pbActions := make([]*pb.Action, 0, len(actions))
	for _, a := range actions {
		pbActions = append(pbActions, &pb.Action{
			Id:          a.ID,
			DetectionId: a.DetectionID,
			ActionType:  a.ActionType,
			DatabaseId:  a.DatabaseID,
			Status:      string(a.Status),
			CreatedAt:   a.CreatedAt.Unix(),
		})
	}

	log.Printf("Retrieved %d pending actions for database: %s", len(actions), req.DatabaseId)

	return &pb.ActionListResponse{
		Actions: pbActions,
	}, nil
}

// ===== [DATABASE OPERATIONS] =====

// RegisterDatabase registers a new database in the knowledge base.
func (s *KnowledgeServer) RegisterDatabase(ctx context.Context, req *pb.RegisterDatabaseRequest) (*pb.DatabaseResponse, error) {
	database := &models.Database{
		ID:               req.DatabaseId,
		ConnectionString: req.ConnectionString,
		DatabaseType:     req.DatabaseType,
		DatabaseName:     req.DatabaseName,
		Host:             req.Host,
		Port:             req.Port,
		Version:          req.Version,
		RegisteredAt:     time.Unix(req.RegisteredAt, 0),
		LastSeen:         time.Now(),
		Status:           "healthy",
		HealthScore:      1.0,
		Metadata:         req.Metadata,
		Enabled:          req.Enabled,
	}

	if err := s.redisClient.RegisterDatabase(ctx, database); err != nil {
		log.Printf("Failed to register database: %v", err)
		return &pb.DatabaseResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Database registered: %s (type: %s, enabled: %v)", database.ID, database.DatabaseType, database.Enabled)

	return &pb.DatabaseResponse{
		Success: true,
		Message: "Database registered successfully",
	}, nil
}

// GetDatabase retrieves database connection info by ID.
func (s *KnowledgeServer) GetDatabase(ctx context.Context, req *pb.GetDatabaseRequest) (*pb.GetDatabaseResponse, error) {
	database, err := s.redisClient.GetDatabase(ctx, req.DatabaseId)
	if err != nil {
		if err.Error() == "failed to get database: redis: nil" {
			log.Printf("Database not found: %s", req.DatabaseId)
			return &pb.GetDatabaseResponse{Found: false}, nil
		}
		log.Printf("Failed to get database: %v", err)
		return &pb.GetDatabaseResponse{Found: false}, nil
	}

	return &pb.GetDatabaseResponse{
		Found:            true,
		DatabaseId:       database.ID,
		ConnectionString: database.ConnectionString,
		DatabaseType:     database.DatabaseType,
		DatabaseName:     database.DatabaseName,
		Host:             database.Host,
		Port:             database.Port,
		Version:          database.Version,
		RegisteredAt:     database.RegisteredAt.Unix(),
		LastSeen:         database.LastSeen.Unix(),
		Status:           database.Status,
		HealthScore:      database.HealthScore,
		Metadata:         database.Metadata,
		Enabled:          database.Enabled,
	}, nil
}

// ListDatabases returns all registered databases.
func (s *KnowledgeServer) ListDatabases(ctx context.Context, req *pb.ListDatabasesRequest) (*pb.DatabaseListResponse, error) {
	databases, err := s.redisClient.ListDatabases(ctx)
	if err != nil {
		log.Printf("Failed to list databases: %v", err)
		return &pb.DatabaseListResponse{}, nil
	}

	pbDatabases := make([]*pb.RegisteredDatabase, 0, len(databases))
	for _, d := range databases {
		if req.EnabledOnly && !d.Enabled {
			continue
		}

		pbDatabases = append(pbDatabases, &pb.RegisteredDatabase{
			DatabaseId:       d.ID,
			DatabaseType:     d.DatabaseType,
			DatabaseName:     d.DatabaseName,
			Host:             d.Host,
			Port:             d.Port,
			Version:          d.Version,
			RegisteredAt:     d.RegisteredAt.Unix(),
			LastSeen:         d.LastSeen.Unix(),
			Status:           d.Status,
			HealthScore:      d.HealthScore,
			Enabled:          d.Enabled,
			ConnectionString: d.ConnectionString,
		})
	}

	log.Printf("Listed %d databases (enabled_only: %v)", len(pbDatabases), req.EnabledOnly)

	return &pb.DatabaseListResponse{
		Databases: pbDatabases,
	}, nil
}

// UpdateDatabase updates an existing database configuration.
func (s *KnowledgeServer) UpdateDatabase(ctx context.Context, req *pb.UpdateDatabaseRequest) (*pb.Response, error) {
	database, err := s.redisClient.GetDatabase(ctx, req.DatabaseId)
	if err != nil {
		log.Printf("Failed to get database for update: %v", err)
		return &pb.Response{
			Success: false,
			Message: "Database not found",
		}, nil
	}

	if req.ConnectionString != "" {
		database.ConnectionString = req.ConnectionString
	}
	if req.DatabaseName != "" {
		database.DatabaseName = req.DatabaseName
	}
	database.Enabled = req.Enabled

	if err := s.redisClient.RegisterDatabase(ctx, database); err != nil {
		log.Printf("Failed to update database: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Database updated: %s (enabled: %v)", req.DatabaseId, req.Enabled)

	return &pb.Response{
		Success: true,
		Message: "Database updated successfully",
	}, nil
}

// UpdateDatabaseHealth updates the health status of a registered database.
func (s *KnowledgeServer) UpdateDatabaseHealth(ctx context.Context, req *pb.UpdateDatabaseHealthRequest) (*pb.Response, error) {
	if err := s.redisClient.UpdateDatabaseHealth(ctx, req.DatabaseId, req.LastSeen, req.Status, req.HealthScore); err != nil {
		log.Printf("Failed to update database health: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Database health updated: %s (status: %s, score: %.2f)", req.DatabaseId, req.Status, req.HealthScore)

	return &pb.Response{
		Success: true,
		Message: "Database health updated successfully",
	}, nil
}

// UnregisterDatabase removes a database from the registry.
func (s *KnowledgeServer) UnregisterDatabase(ctx context.Context, req *pb.UnregisterDatabaseRequest) (*pb.Response, error) {
	if err := s.redisClient.UnregisterDatabase(ctx, req.DatabaseId); err != nil {
		log.Printf("Failed to unregister database: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Database unregistered: %s", req.DatabaseId)

	return &pb.Response{
		Success: true,
		Message: "Database unregistered successfully",
	}, nil
}

// ===== [SYSTEM STATISTICS] =====

// GetSystemStats returns system-wide statistics.
func (s *KnowledgeServer) GetSystemStats(ctx context.Context, req *pb.GetSystemStatsRequest) (*pb.GetSystemStatsResponse, error) {
	databases, _ := s.redisClient.ListDatabases(ctx)
	totalDatabases := int32(len(databases))

	var healthyCount, degradedCount, offlineCount int32
	for _, db := range databases {
		switch db.Status {
		case "healthy":
			healthyCount++
		case "degraded":
			degradedCount++
		case "offline":
			offlineCount++
		}
	}

	activeDetections, _ := s.redisClient.CountAllActiveDetections(ctx)

	queuedCount, _ := s.redisClient.CountActionsByStatus(ctx, models.StatusQueued)
	executingCount, _ := s.redisClient.CountActionsByStatus(ctx, models.StatusExecuting)
	completedCount, _ := s.redisClient.CountActionsByStatus(ctx, models.StatusCompleted)
	failedCount, _ := s.redisClient.CountActionsByStatus(ctx, models.StatusFailed)

	totalActions, _ := s.redisClient.CountAllActions(ctx)

	uptime := int64(time.Since(s.startTime).Seconds())

	return &pb.GetSystemStatsResponse{
		TotalDatabases:     totalDatabases,
		HealthyDatabases:   healthyCount,
		DegradedDatabases:  degradedCount,
		OfflineDatabases:   offlineCount,
		TotalDetections:    activeDetections,
		ActiveDetections:   activeDetections,
		ResolvedDetections: 0,
		TotalActions:       totalActions,
		ActionsQueued:      queuedCount,
		ActionsExecuting:   executingCount,
		ActionsCompleted:   completedCount,
		ActionsFailed:      failedCount,
		UptimeSeconds:      uptime,
	}, nil
}

// ===== [CONFIGURATION MANAGEMENT] =====

// GetSystemConfig retrieves the system configuration.
func (s *KnowledgeServer) GetSystemConfig(ctx context.Context, req *pb.GetSystemConfigRequest) (*pb.SystemConfig, error) {
	config, err := s.redisClient.GetSystemConfig(ctx)
	if err != nil {
		log.Printf("Failed to get system config: %v", err)
		return &pb.SystemConfig{
			Thresholds: &pb.DetectionThresholds{
				ConnectionPoolCritical:  0.8,
				SequentialScanThreshold: 1000,
				SequentialScanDelta:     100.0,
				P95LatencyMs:            100.0,
				CacheHitRateThreshold:   0.9,
			},
			OnboardingComplete: false,
		}, nil
	}

	return config, nil
}

// SaveSystemConfig saves the system configuration.
func (s *KnowledgeServer) SaveSystemConfig(ctx context.Context, req *pb.SaveSystemConfigRequest) (*pb.Response, error) {
	if req.Config == nil {
		return &pb.Response{
			Success: false,
			Message: "Config cannot be nil",
		}, nil
	}

	if err := s.redisClient.SaveSystemConfig(ctx, req.Config); err != nil {
		log.Printf("Failed to save system config: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("System config saved (onboarding_complete: %v)", req.Config.OnboardingComplete)

	return &pb.Response{
		Success: true,
		Message: "System configuration saved successfully",
	}, nil
}

// GetSystemStatus returns the current system status.
func (s *KnowledgeServer) GetSystemStatus(ctx context.Context, req *pb.GetSystemStatusRequest) (*pb.SystemStatus, error) {
	config, _ := s.redisClient.GetSystemConfig(ctx)
	databases, _ := s.redisClient.ListDatabases(ctx)

	hasEnabledDB := false
	for _, db := range databases {
		if db.Enabled {
			hasEnabledDB = true
			break
		}
	}

	onboardingComplete := false
	if config != nil {
		onboardingComplete = config.OnboardingComplete
	}

	serviceStates := make(map[string]string)

	return &pb.SystemStatus{
		Configured:         hasEnabledDB,
		OnboardingComplete: onboardingComplete,
		ServiceStates:      serviceStates,
	}, nil
}

// FlushAllData clears all data from Redis.
func (s *KnowledgeServer) FlushAllData(ctx context.Context, req *pb.FlushAllDataRequest) (*pb.FlushAllDataResponse, error) {
	if err := s.redisClient.FlushAll(ctx); err != nil {
		log.Printf("Failed to flush all data: %v", err)
		return &pb.FlushAllDataResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("All data flushed from Redis")

	return &pb.FlushAllDataResponse{
		Success: true,
		Message: "All data flushed successfully",
	}, nil
}
