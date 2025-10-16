package grpcserver

import (
	"context"
	"io"
	"log"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type MetricsServer struct {
	pb.UnimplementedMetricsServiceServer
}

func NewMetricsServer() *MetricsServer {
	return &MetricsServer{}
}

func (s *MetricsServer) StreamMetrics(stream pb.MetricsService_StreamMetricsServer) error {
	log.Println("Client connected, waiting for metrics stream...")

	metricsCount := int64(0)

	for {
		metric, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Stream closed. Total metrics received: %d", metricsCount)

			return stream.SendAndClose(&pb.MetricsAck{
				TotalMetrics: metricsCount,
				Status:       "healthy",
			})
		}
		if err != nil {
			log.Printf("Error receiving metric: %v", err)
			return err
		}

		// Log the received metric
		metricsCount++
		log.Printf("Metric #%d received:", metricsCount)
		log.Printf("DTimestamp: %d", metric.Timestamp)
		log.Printf("Active Connections: %d", metric.ActiveConnections)
		log.Printf("Max Connections: %d", metric.MaxConnections)
		log.Printf("Cache Hit Rate: %.2f%%", metric.CacheHitRate*100)
	}
}

func (s *MetricsServer) RegisterDatabase(ctx context.Context, info *pb.DatabaseInfo) (*pb.RegistrationAck, error) {
	log.Printf("Database registered: %s (%s)", info.DatabaseName, info.DatabaseType)

	return &pb.RegistrationAck{
		Success:    true,
		Message:    "Database registered successfully",
		AssignedId: info.DatabaseId,
	}, nil
}
