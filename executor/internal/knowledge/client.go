package knowledge

import (
	"context"
	"fmt"
	"log"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.KnowledgeServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to knowledge service at: %s, %w", addr, err)
	}

	log.Printf("Executor Connected to Brain at: %s", addr)

	return &Client{
		conn:   conn,
		client: pb.NewKnowledgeServiceClient(conn),
	}, nil
}

func (k *Client) RegisterAction(ctx context.Context, req *pb.RegisterActionRequest) error {
	resp, err := k.client.RegisterAction(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to register action: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("knowledge rejected action registration: %s", resp.Message)
	}

	return nil
}

func (k *Client) UpdateActionStatus(ctx context.Context, req *pb.UpdateActionRequest) error {
	resp, err := k.client.UpdateActionStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update action status: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("knowledge rejected status update: %s", resp.Message)
	}

	return nil
}

func (k *Client) GetPendingActions(ctx context.Context, databaseID string) ([]*pb.Action, error) {
	resp, err := k.client.GetPendingActions(ctx, &pb.DatabaseFilterRequest{
		DatabaseId: databaseID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pending actions: %w", err)
	}

	return resp.Actions, nil
}

// GetSystemConfig fetches the system configuration from Knowledge service.
func (c *Client) GetSystemConfig(ctx context.Context) (*pb.SystemConfig, error) {
	resp, err := c.client.GetSystemConfig(ctx, &pb.GetSystemConfigRequest{})
	if err != nil {
		return nil, fmt.Errorf("GetSystemConfig RPC failed: %w", err)
	}
	return resp, nil
}

// GetExecutionMode fetches just the execution mode, with default fallback.
func (c *Client) GetExecutionMode(ctx context.Context) string {
	config, err := c.GetSystemConfig(ctx)
	if err != nil {
		log.Printf("Warning: failed to get execution mode, defaulting to autonomous: %v", err)
		return "autonomous"
	}

	if config.ExecutionMode == "" {
		return "autonomous" // Default for backwards compatibility
	}

	return config.ExecutionMode
}

func (k *Client) GetServiceClient() pb.KnowledgeServiceClient {
	return k.client
}

func (k *Client) Close() error {
	if k.conn != nil {
		return k.conn.Close()
	}
	return nil
}
