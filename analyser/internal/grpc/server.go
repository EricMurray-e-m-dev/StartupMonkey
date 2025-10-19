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
		snapshot, err := stream.Recv()
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

		// Log the received normalized metrics
		metricsCount++
		log.Printf("Metric #%d received:", metricsCount)
		log.Printf("  Database: %s (%s)", snapshot.DatabaseId, snapshot.DatabaseType)
		log.Printf("  Timestamp: %d", snapshot.Timestamp)

		// Health scores
		log.Printf("  Health Score: %.2f", snapshot.HealthScore)
		log.Printf("  Connection Health: %.2f", snapshot.ConnectionHealth)
		log.Printf("  Query Health: %.2f", snapshot.QueryHealth)
		log.Printf("  Storage Health: %.2f", snapshot.StorageHealth)
		log.Printf("  Cache Health: %.2f", snapshot.CacheHealth)

		// Available metrics
		log.Printf("  Available Metrics: %v", snapshot.AvailableMetrics)

		// Log raw measurements if available
		if snapshot.Measurements != nil {
			log.Printf("  Raw Measurements:")

			if snapshot.Measurements.ActiveConnections != nil {
				log.Printf("    Active Connections: %d", *snapshot.Measurements.ActiveConnections)
			}

			if snapshot.Measurements.MaxConnections != nil {
				log.Printf("    Max Connections: %d", *snapshot.Measurements.MaxConnections)
			}

			if snapshot.Measurements.CacheHitRate != nil {
				log.Printf("    Cache Hit Rate: %.2f%%", *snapshot.Measurements.CacheHitRate*100)
			}

			if snapshot.Measurements.P95QueryLatencyMs != nil {
				log.Printf("    P95 Query Latency: %.2fms", *snapshot.Measurements.P95QueryLatencyMs)
			}
		}
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
