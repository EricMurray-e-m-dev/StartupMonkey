package knowledge

import (
	"context"
	"fmt"
	"log"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type KnowledgeClient struct {
	conn   *grpc.ClientConn
	client pb.KnowledgeServiceClient
}

func NewKnowledgeClient(addr string) (*KnowledgeClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Knowledge service at %s: %w", addr, err)
	}

	log.Printf("Connected to Knowledge service at %s", addr)

	return &KnowledgeClient{
		conn:   conn,
		client: pb.NewKnowledgeServiceClient(conn),
	}, nil
}

func (k *KnowledgeClient) IsDetectionActive(ctx context.Context, key string) (bool, error) {
	resp, err := k.client.IsDetectionActive(ctx, &pb.DetectionKeyRequest{
		Key: key,
	})
	if err != nil {
		return false, err
	}

	return resp.IsActive, nil
}

func (k *KnowledgeClient) RegisterDetection(ctx context.Context, detection *models.Detection) error {
	_, err := k.client.RegisterDetection(ctx, &pb.RegisterDetectionRequest{
		Id:         detection.ID,
		Key:        detection.Key,
		Severity:   string(detection.Severity),
		Category:   string(detection.Category),
		DatabaseId: detection.DatabaseID,
		Value:      0, // TODO: Extract meaningful value from Evidence
		CreatedAt:  detection.Timestamp,
	})

	if err != nil {
		return fmt.Errorf("failed to register detection with Knowledge: %w", err)
	}

	return nil
}

func (k *KnowledgeClient) MarkDetectionResolved(ctx context.Context, detectionID string, solution string) error {
	_, err := k.client.MarkDetectionResolved(ctx, &pb.ResolveDetectionRequest{
		DetectionId: detectionID,
		Solution:    solution,
	})

	if err != nil {
		return fmt.Errorf("failed to mark detection resolved: %w", err)
	}

	log.Printf("Detection marked as resolved in Knowledge: %s (solution: %s)", detectionID, solution)

	return nil
}

// GetSystemConfig fetches the system configuration from Knowledge service.
func (k *KnowledgeClient) GetSystemConfig(ctx context.Context) (*pb.SystemConfig, error) {
	resp, err := k.client.GetSystemConfig(ctx, &pb.GetSystemConfigRequest{})
	if err != nil {
		return nil, fmt.Errorf("GetSystemConfig RPC failed: %w", err)
	}

	return resp, nil
}

func (k *KnowledgeClient) Close() error {
	if k.conn != nil {
		return k.conn.Close()
	}
	return nil
}
