package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/eventbus"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Orchestrator struct {
	config          *config.Config
	adapter         adapter.MetricAdapter
	normaliser      normaliser.Normaliser
	client          *grpcclient.MetricsClient
	natsPublisher   *eventbus.Publisher
	knowledgeClient pb.KnowledgeServiceClient
}

func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

func (o *Orchestrator) Start() error {
	log.Printf("Starting Collector Orchestrator...")

	var err error
	o.adapter, err = adapter.NewAdapter(o.config.DBAdapter, o.config.DBConnectionString, o.config.DatabaseID)
	if err != nil {
		return err
	}

	log.Printf("Connecting to database (%s)", o.config.DBAdapter)
	if err := o.adapter.Connect(); err != nil {
		return err
	}
	log.Printf("Database Connected.")

	if err := o.adapter.HealthCheck(); err != nil {
		return err
	}
	log.Printf("Database healthy.")

	o.normaliser = normaliser.NewNormaliser(o.config.DBAdapter)
	log.Printf("Normaliser created for %s", o.config.DBAdapter)

	o.client = grpcclient.NewMetricsClient(o.config.AnalyserAddress)
	if err := o.client.Connect(); err != nil {
		return err
	}
	log.Printf("Connected to Analyser at %s", o.config.AnalyserAddress)

	if o.config.NatsURL != "" {
		o.natsPublisher, err = eventbus.NewPublisher(o.config.NatsURL)
		if err != nil {
			log.Printf("Warning: failed to connect to NATS, Dashboard UI live metrics will be unavailable")
		}
	}

	log.Printf("Connecting to brain at: %s", o.config.KnowledgeAddress)
	knowledgeConn, err := grpc.NewClient(
		o.config.KnowledgeAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Printf("warning failed to connect to brain: %v", err)
	} else {
		o.knowledgeClient = pb.NewKnowledgeServiceClient(knowledgeConn)
		log.Printf("Connected to brain")

		if err := o.registerDatabase(); err != nil {
			log.Printf("warning failed to register database to brain: %v", err)
		} else {
			log.Printf("Database registered with brain")
		}
	}

	return nil
}

func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting collection (interval: %v)", o.config.CollectionInterval)

	ticker := time.NewTicker(o.config.CollectionInterval)
	defer ticker.Stop()

	if err := o.collectAndSend(ctx); err != nil {
		log.Printf("Error in collection cycle: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutting down collection")
			return ctx.Err()

		case <-ticker.C:
			if err := o.collectAndSend(ctx); err != nil {
				log.Printf("Error in collection cycle: %v", err)
			}
		}
	}
}

func (o *Orchestrator) collectAndSend(ctx context.Context) error {
	log.Printf("--- Collection Cycle Start ----")

	log.Printf("Collecting Metrics from database...")
	rawMetrics, err := o.adapter.CollectMetrics()
	if err != nil {
		return err
	}

	log.Printf("Normalising metrics...")
	normalised, err := o.normaliser.Normalise(rawMetrics)
	if err != nil {
		return err
	}

	log.Printf("Metrics normalised - Health Score: %.2f, Available: %v", normalised.HealthScore, normalised.AvailableMetrics)

	snapshot := o.toProtobuf(normalised)

	log.Printf("Sending metrics to analyser....")
	ack, err := o.client.StreamMetrics(ctx, []*pb.MetricSnapshot{snapshot})
	if err != nil {
		return err
	}

	log.Printf("Metrics sent successfully. Ack: %d metrics, status: %s", ack.TotalMetrics, ack.Status)

	if o.natsPublisher != nil {
		err := o.natsPublisher.PublishMetrics(normalised)
		if err != nil {
			log.Printf("Warning: failed to publish metrics to nats")
		}
	}
	log.Printf("--- Collection Cycle Complete ---")

	return nil

}

func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	if o.client != nil {
		if err := o.client.Close(); err != nil {
			log.Printf("Error closing gRPC Client: %v", err)
		}
	}

	if o.natsPublisher != nil {
		o.natsPublisher.Close()
	}

	if o.adapter != nil {
		if err := o.adapter.Close(); err != nil {
			log.Printf("Error closing database adapter: %v", err)
		}
	}

	log.Printf("Orchestator stopped")
	return nil

}

func (o *Orchestrator) toProtobuf(n *normaliser.NormalisedMetrics) *pb.MetricSnapshot {
	snapshot := &pb.MetricSnapshot{
		// Metadata
		DatabaseId:   n.DatabaseID,
		DatabaseType: n.DatabaseType,
		Timestamp:    n.Timestamp,

		// Health scores
		HealthScore:      n.HealthScore,
		ConnectionHealth: n.ConnectionHealth,
		QueryHealth:      n.QueryHealth,
		StorageHealth:    n.StorageHealth,
		CacheHealth:      n.CacheHealth,

		// Context
		AvailableMetrics: n.AvailableMetrics,

		MetricDeltas:     n.MetricDeltas,
		TimeDeltaSeconds: &n.TimeDeltaSeconds,

		// Extended metrics
		ExtendedMetrics: n.ExtendedMetrics,
		Labels:          n.Labels,

		// Measurements
		Measurements: &pb.Measurements{
			// Connections
			ActiveConnections:  n.Measurements.ActiveConnections,
			IdleConnections:    n.Measurements.IdleConnections,
			MaxConnections:     n.Measurements.MaxConnections,
			WaitingConnections: n.Measurements.WaitingConnections,

			// Queries
			AvgQueryLatencyMs: n.Measurements.AvgQueryLatencyMs,
			P50QueryLatencyMs: n.Measurements.P50QueryLatencyMs,
			P95QueryLatencyMs: n.Measurements.P95QueryLatencyMs,
			P99QueryLatencyMs: n.Measurements.P99QueryLatencyMs,
			SlowQueryCount:    n.Measurements.SlowQueryCount,
			SequentialScans:   n.Measurements.SequentialScans,

			// Storage
			UsedStorageBytes:  n.Measurements.UsedStorageBytes,
			TotalStorageBytes: n.Measurements.TotalStorageBytes,
			FreeStorageBytes:  n.Measurements.FreeStorageBytes,

			// Cache
			CacheHitRate:   n.Measurements.CacheHitRate,
			CacheHitCount:  n.Measurements.CacheHitCount,
			CacheMissCount: n.Measurements.CacheMissCount,
		},
	}

	return snapshot
}

// registerDatabase registers the database with Knowledge service
func (o *Orchestrator) registerDatabase() error {
	// Parse connection string to extract host, port, and database name
	host, port, _ := o.parseConnectionString(o.config.DBConnectionString)

	req := &pb.RegisterDatabaseRequest{
		DatabaseId:       o.config.DatabaseID,
		ConnectionString: o.config.DBConnectionString,
		DatabaseType:     o.config.DBAdapter,
		DatabaseName:     o.config.DatabaseName,
		Host:             host,
		Port:             port,
		Version:          "unknown", // TODO: Query database for version
		RegisteredAt:     time.Now().Unix(),
		Metadata:         map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := o.knowledgeClient.RegisterDatabase(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to register database: %s", resp.Message)
	}

	log.Printf("Database registered: %s (%s)", o.config.DatabaseID, o.config.DBAdapter)
	return nil
}

// parseConnectionString extracts host and port from connection string
func (o *Orchestrator) parseConnectionString(connStr string) (string, int32, string) {
	host := "localhost"
	port := int32(5432) // Default for PostgreSQL

	// Set default port based on database type
	switch o.config.DBAdapter {
	case "postgres", "postgresql":
		port = 5432
	case "mysql":
		port = 3306
	case "mongodb":
		port = 27017
	}

	// Parse connection string
	if strings.Contains(connStr, "://") {
		u, err := url.Parse(connStr)
		if err == nil {
			host = u.Hostname()
			if u.Port() != "" {
				if p, err := strconv.Atoi(u.Port()); err == nil {
					port = int32(p)
				}
			}
		}
	}

	return host, port, "unknown"
}
