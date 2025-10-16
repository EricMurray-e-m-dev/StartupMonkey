package grpc

import (
	"context"
	"fmt"
	"log"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MetricsClient struct {
	analyserAddress string
	conn            *grpc.ClientConn
	client          pb.MetricsServiceClient
}

func NewMetricsClient(analyserAddress string) *MetricsClient {
	return &MetricsClient{
		analyserAddress: analyserAddress,
	}
}

func (c *MetricsClient) Connect() error {

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

func (c *MetricsClient) StreamMetrics(ctx context.Context, metrics []*pb.MetricsSnapshot) (*pb.MetricsAck, error) {

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
		log.Printf("Sent metric for DB: %s", metric.DatabaseId)
	}

	ack, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive ack: %w", err)
	}

	log.Printf("Received ack: %d metrics , status %s", ack.TotalMetrics, ack.Status)
	return ack, nil
}

func (c *MetricsClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()

		c.conn = nil
		c.client = nil

		return err
	}

	return nil
}
