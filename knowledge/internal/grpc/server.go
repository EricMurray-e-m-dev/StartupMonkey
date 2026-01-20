package grpc

import (
	"context"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type KnowledgeServer struct {
	pb.UnimplementedKnowledgeServiceServer
	redisClient *redis.Client
}

func NewKnowledgeServer(redisClient *redis.Client) *KnowledgeServer {
	return &KnowledgeServer{
		redisClient: redisClient,
	}
}

// RegisterDetection registers a new detection in the knowledge base
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
		TTL:        0, // No TTL for active detections
	}

	if err := s.redisClient.RegisterDetection(ctx, detection); err != nil {
		log.Printf("Failed to register detection: %v", err)
		return &pb.DetectionResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Detection registered: %s (key: %s)", detection.ID, detection.Key)

	return &pb.DetectionResponse{
		Success:     true,
		Message:     "Detection registered successfully",
		DetectionId: detection.ID,
	}, nil
}

// IsDetectionActive checks if a detection with the given key is active
func (s *KnowledgeServer) IsDetectionActive(ctx context.Context, req *pb.DetectionKeyRequest) (*pb.DetectionStatusResponse, error) {
	isActive, err := s.redisClient.IsDetectionActive(ctx, req.Key)
	if err != nil {
		log.Printf("Failed to check detection status: %v", err)
		return &pb.DetectionStatusResponse{
			IsActive: false,
		}, err
	}

	// Get detection ID if active
	detectionID := ""
	if isActive {
		keyMapping := "detection_key:" + req.Key
		detectionID, _ = s.redisClient.GetClient().Get(ctx, keyMapping).Result()
	}

	return &pb.DetectionStatusResponse{
		IsActive:    isActive,
		DetectionId: detectionID,
	}, nil
}

// GetActiveDetections returns all active detections for a database
func (s *KnowledgeServer) GetActiveDetections(ctx context.Context, req *pb.DatabaseFilterRequest) (*pb.DetectionListResponse, error) {
	detections, err := s.redisClient.GetActiveDetections(ctx, req.DatabaseId)
	if err != nil {
		log.Printf("Failed to get active detections: %v", err)
		return &pb.DetectionListResponse{
			Detections: []*pb.Detection{},
		}, err
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

// MarkDetectionResolved marks a detection as resolved
func (s *KnowledgeServer) MarkDetectionResolved(ctx context.Context, req *pb.ResolveDetectionRequest) (*pb.Response, error) {
	if err := s.redisClient.MarkDetectionResolved(ctx, req.DetectionId, req.Solution); err != nil {
		log.Printf("Failed to mark detection resolved: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Detection resolved: %s (solution: %s)", req.DetectionId, req.Solution)

	return &pb.Response{
		Success: true,
		Message: "Detection marked as resolved",
	}, nil
}

// ===== [ACTIONS OPERATIONS] =====

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
		}, err
	}

	log.Printf("Action registered: %s (type: %s, detection: %s)", action.ID, action.ActionType, action.DetectionID)

	return &pb.ActionResponse{
		Success:  true,
		Message:  "Action successfully registered",
		ActionId: action.ID,
	}, nil
}

func (s *KnowledgeServer) UpdateActionStatus(ctx context.Context, req *pb.UpdateActionRequest) (*pb.Response, error) {
	if err := s.redisClient.UpdateActionStatus(ctx, req.ActionId, models.ActionStatus(req.Status), req.Message, req.Error); err != nil {
		log.Printf("Failed to update action status: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Action status updated %s => %s", req.ActionId, req.Status)

	return &pb.Response{
		Success: true,
		Message: "Action status updated successfully",
	}, nil
}

func (s *KnowledgeServer) GetPendingActions(ctx context.Context, req *pb.DatabaseFilterRequest) (*pb.ActionListResponse, error) {
	actions, err := s.redisClient.GetPendingActions(ctx, req.DatabaseId)
	if err != nil {
		log.Printf("Failed to get pending actions: %v", err)
		return &pb.ActionListResponse{
			Actions: []*pb.Action{},
		}, err
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

// RegisterDatabase registers a new database in the knowledge base
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
	}

	if err := s.redisClient.RegisterDatabase(ctx, database); err != nil {
		log.Printf("Failed to register database: %v", err)
		return &pb.DatabaseResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Database registered: %s (type: %s)", database.ID, database.DatabaseType)

	return &pb.DatabaseResponse{
		Success: true,
		Message: "Database registered successfully",
	}, nil
}

// GetDatabase retrieves database connection info
func (s *KnowledgeServer) GetDatabase(ctx context.Context, req *pb.GetDatabaseRequest) (*pb.GetDatabaseResponse, error) {
	database, err := s.redisClient.GetDatabase(ctx, req.DatabaseId)
	if err != nil {
		if err.Error() == "failed to get database: redis: nil" {
			log.Printf("Database not found: %s", req.DatabaseId)
			return &pb.GetDatabaseResponse{Found: false}, nil
		}
		log.Printf("Failed to get database: %v", err)
		return &pb.GetDatabaseResponse{Found: false}, err
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
	}, nil
}

// ListDatabases returns all registered databases
func (s *KnowledgeServer) ListDatabases(ctx context.Context, req *pb.ListDatabasesRequest) (*pb.DatabaseListResponse, error) {
	databases, err := s.redisClient.ListDatabases(ctx)
	if err != nil {
		log.Printf("Failed to list databases: %v", err)
		return &pb.DatabaseListResponse{}, err
	}

	pbDatabases := make([]*pb.RegisteredDatabase, 0, len(databases))
	for _, d := range databases {
		pbDatabases = append(pbDatabases, &pb.RegisteredDatabase{
			DatabaseId:   d.ID,
			DatabaseType: d.DatabaseType,
			DatabaseName: d.DatabaseName,
			Host:         d.Host,
			Port:         d.Port,
			Version:      d.Version,
			RegisteredAt: d.RegisteredAt.Unix(),
			LastSeen:     d.LastSeen.Unix(),
			Status:       d.Status,
			HealthScore:  d.HealthScore,
		})
	}

	log.Printf("Listed %d databases", len(databases))

	return &pb.DatabaseListResponse{
		Databases: pbDatabases,
	}, nil
}

// UpdateDatabaseHealth updates health status
func (s *KnowledgeServer) UpdateDatabaseHealth(ctx context.Context, req *pb.UpdateDatabaseHealthRequest) (*pb.Response, error) {
	if err := s.redisClient.UpdateDatabaseHealth(ctx, req.DatabaseId, req.LastSeen, req.Status, req.HealthScore); err != nil {
		log.Printf("Failed to update database health: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Database health updated: %s (status: %s, score: %.2f)", req.DatabaseId, req.Status, req.HealthScore)

	return &pb.Response{
		Success: true,
		Message: "Database health updated successfully",
	}, nil
}

// UnregisterDatabase removes a database
func (s *KnowledgeServer) UnregisterDatabase(ctx context.Context, req *pb.UnregisterDatabaseRequest) (*pb.Response, error) {
	if err := s.redisClient.UnregisterDatabase(ctx, req.DatabaseId); err != nil {
		log.Printf("Failed to unregister database: %v", err)
		return &pb.Response{
			Success: false,
			Message: err.Error(),
		}, err
	}

	log.Printf("Database unregistered: %s", req.DatabaseId)

	return &pb.Response{
		Success: true,
		Message: "Database unregistered successfully",
	}, nil
}

// GetSystemStats returns system-wide statistics
func (s *KnowledgeServer) GetSystemStats(ctx context.Context, req *pb.GetSystemStatsRequest) (*pb.GetSystemStatsResponse, error) {
	// Count total databases
	databases, _ := s.redisClient.ListDatabases(ctx)
	totalDatabases := int32(len(databases))

	// Count by status
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

	// Count detections and actions
	// TODO: Implement proper counting across all databases
	totalDetections := int32(0)
	activeDetections := int32(0)
	totalActions := int32(0)
	queuedCount := int32(0)
	executingCount := int32(0)
	completedCount := int32(0)
	failedCount := int32(0)

	return &pb.GetSystemStatsResponse{
		TotalDatabases:     totalDatabases,
		HealthyDatabases:   healthyCount,
		DegradedDatabases:  degradedCount,
		OfflineDatabases:   offlineCount,
		TotalDetections:    totalDetections,
		ActiveDetections:   activeDetections,
		ResolvedDetections: totalDetections - activeDetections,
		TotalActions:       totalActions,
		ActionsQueued:      queuedCount,
		ActionsExecuting:   executingCount,
		ActionsCompleted:   completedCount,
		ActionsFailed:      failedCount,
		UptimeSeconds:      0,
	}, nil
}
