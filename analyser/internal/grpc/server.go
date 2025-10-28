package grpcserver

import (
	"context"
	"io"
	"log"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/eventbus"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type MetricsServer struct {
	pb.UnimplementedMetricsServiceServer
	engine    *engine.Engine
	publisher *eventbus.Publisher
}

func NewMetricsServer(eng *engine.Engine, pub *eventbus.Publisher) *MetricsServer {
	return &MetricsServer{
		engine:    eng,
		publisher: pub,
	}
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

		normalised := s.toNormalisedMetrics(snapshot)

		detections := s.engine.RunDetectors(normalised)

		if len(detections) > 0 {
			log.Printf("Found %d issues in database: %s", len(detections), snapshot.DatabaseId)

			for _, detection := range detections {
				log.Printf("\t[%s] %s", detection.Severity, detection.Title)
				log.Printf("\t%s", detection.Description)
				log.Printf("\tRecommendation: %s", detection.Recommendation)

				if err := s.publisher.PublishDetection(detection); err != nil {
					log.Printf("\tFailed to publish detection event: %v", err)
				} else {
					log.Printf("\tPublished to event bus")
				}
			}
		} else {
			log.Printf("No issues detected in database: %s", snapshot.DatabaseId)
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

func (s *MetricsServer) toNormalisedMetrics(snapshot *pb.MetricSnapshot) *normaliser.NormalisedMetrics {
	normalised := &normaliser.NormalisedMetrics{
		DatabaseID:   snapshot.DatabaseId,
		DatabaseType: snapshot.DatabaseType,
		Timestamp:    snapshot.Timestamp,

		HealthScore:      snapshot.HealthScore,
		ConnectionHealth: snapshot.ConnectionHealth,
		QueryHealth:      snapshot.QueryHealth,
		StorageHealth:    snapshot.StorageHealth,
		CacheHealth:      snapshot.CacheHealth,

		AvailableMetrics: snapshot.AvailableMetrics,
		ExtendedMetrics:  snapshot.ExtendedMetrics,

		Measurements: normaliser.Measurements{},
	}

	if snapshot.Measurements != nil {
		normalised.Measurements = normaliser.Measurements{
			ActiveConnections:  snapshot.Measurements.ActiveConnections,
			IdleConnections:    snapshot.Measurements.IdleConnections,
			MaxConnections:     snapshot.Measurements.MaxConnections,
			WaitingConnections: snapshot.Measurements.WaitingConnections,

			AvgQueryLatencyMs: snapshot.Measurements.AvgQueryLatencyMs,
			P50QueryLatencyMs: snapshot.Measurements.P50QueryLatencyMs,
			P95QueryLatencyMs: snapshot.Measurements.P95QueryLatencyMs,
			P99QueryLatencyMs: snapshot.Measurements.P99QueryLatencyMs,
			SlowQueryCount:    snapshot.Measurements.SlowQueryCount,
			SequentialScans:   snapshot.Measurements.SequentialScans,

			UsedStorageBytes:  snapshot.Measurements.UsedStorageBytes,
			TotalStorageBytes: snapshot.Measurements.TotalStorageBytes,
			FreeStorageBytes:  snapshot.Measurements.FreeStorageBytes,

			CacheHitRate:   snapshot.Measurements.CacheHitRate,
			CacheHitCount:  snapshot.Measurements.CacheHitCount,
			CacheMissCount: snapshot.Measurements.CacheMissCount,
		}
	}

	return normalised
}
