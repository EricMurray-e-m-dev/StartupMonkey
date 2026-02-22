// Package grpc provides the gRPC client for communicating with the Analyser service.
package grpc

import (
	"context"
	"fmt"
	"log"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MetricsClient handles streaming metrics to the Analyser service.
type MetricsClient struct {
	analyserAddress string
	conn            *grpc.ClientConn
	client          pb.MetricsServiceClient
}

// NewMetricsClient creates a new MetricsClient for the given Analyser address.
func NewMetricsClient(analyserAddress string) *MetricsClient {
	return &MetricsClient{
		analyserAddress: analyserAddress,
	}
}

// Connect establishes a gRPC connection to the Analyser service.
func (c *MetricsClient) Connect() error {
	if c.analyserAddress == "" {
		return fmt.Errorf("analyser address cannot be empty")
	}

	conn, err := grpc.NewClient(
		c.analyserAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	c.conn = conn
	c.client = pb.NewMetricsServiceClient(conn)

	log.Printf("Connected to Analyser: %s", c.analyserAddress)
	return nil
}

// StreamMetrics sends a batch of metric snapshots to the Analyser.
func (c *MetricsClient) StreamMetrics(ctx context.Context, metrics []*pb.MetricSnapshot) (*pb.MetricsAck, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	stream, err := c.client.StreamMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	for _, metric := range metrics {
		if err := stream.Send(metric); err != nil {
			return nil, fmt.Errorf("failed to send metric: %w", err)
		}
	}

	ack, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive ack: %w", err)
	}

	return ack, nil
}

// Close closes the gRPC connection.
func (c *MetricsClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}
