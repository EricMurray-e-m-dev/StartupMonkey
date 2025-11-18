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

	log.Printf("✅ Detection resolved: %s (solution: %s)", req.DetectionId, req.Solution)

	return &pb.Response{
		Success: true,
		Message: "Detection marked as resolved",
	}, nil
}

// Stub implementations for Action methods (Issue #58)
func (s *KnowledgeServer) RegisterAction(ctx context.Context, req *pb.RegisterActionRequest) (*pb.ActionResponse, error) {
	return &pb.ActionResponse{
		Success:  true,
		Message:  "Action registration not yet implemented",
		ActionId: req.Id,
	}, nil
}

func (s *KnowledgeServer) UpdateActionStatus(ctx context.Context, req *pb.UpdateActionRequest) (*pb.Response, error) {
	return &pb.Response{
		Success: true,
		Message: "Action update not yet implemented",
	}, nil
}

func (s *KnowledgeServer) GetPendingActions(ctx context.Context, req *pb.DatabaseFilterRequest) (*pb.ActionListResponse, error) {
	return &pb.ActionListResponse{
		Actions: []*pb.Action{},
	}, nil
}
