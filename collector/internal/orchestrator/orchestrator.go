package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/eventbus"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

// Orchestrator manages the Collector service lifecycle and coordinates
// metric collection, normalization, and distribution to downstream services.
//
// Lifecycle:
//  1. Start() - Initializes connections to database, Analyser, NATS, and Knowledge
//  2. Run() - Begins periodic metric collection on configured interval
//  3. Stop() - Gracefully closes all connections and resources
//
// The orchestrator implements graceful degradation:
//   - NATS failure: Metrics still sent to Analyser (Dashboard unavailable)
//   - Knowledge failure: Database not registered (Executor cannot deploy actions)
type Orchestrator struct {
	config *config.Config

	// Database connection and metric collection
	adapter    adapter.MetricAdapter
	normaliser normaliser.Normaliser

	// Downstream service connections
	client          *grpcclient.MetricsClient // gRPC to Analyser
	natsPublisher   *eventbus.Publisher       // NATS event bus for Dashboard
	knowledgeClient *knowledge.Client         // Knowledge service client
}

// NewOrchestrator creates a new Orchestrator instance with the provided configuration.
// The orchestrator is not started until Start() is called.
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

// Start initializes all service connections and prepares the orchestrator for metric collection.
// This method must be called before Run().
//
// Start connects to:
//   - Target database (required)
//   - Analyser service via gRPC (required)
//   - NATS event bus (optional - for Dashboard real-time metrics)
//   - Knowledge service via gRPC (optional - for database registration)
//
// Returns an error if any required connection fails.
func (o *Orchestrator) Start() error {
	log.Printf("Starting Collector Orchestrator...")

	// Initialize database connection
	if err := o.connectDatabase(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize downstream services
	if err := o.connectAnalyser(); err != nil {
		return fmt.Errorf("failed to connect to Analyser: %w", err)
	}

	o.connectNATS()      // Optional - warnings logged on failure
	o.connectKnowledge() // Optional - warnings logged on failure

	log.Printf("Collector Orchestrator started successfully")
	return nil
}

// connectDatabase establishes connection to the target database and performs health check.
func (o *Orchestrator) connectDatabase() error {
	log.Printf("Connecting to database (adapter: %s, id: %s)", o.config.DBAdapter, o.config.DatabaseID)

	var err error
	o.adapter, err = adapter.NewAdapter(o.config.DBAdapter, o.config.DBConnectionString, o.config.DatabaseID)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := o.adapter.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	if err := o.adapter.HealthCheck(); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	o.normaliser = normaliser.NewNormaliser(o.config.DBAdapter)
	log.Printf("Database connected and healthy")
	return nil
}

// connectAnalyser establishes gRPC connection to the Analyser service.
// This is a required connection - failure will prevent metric streaming.
func (o *Orchestrator) connectAnalyser() error {
	log.Printf("Connecting to Analyser at: %s", o.config.AnalyserAddress)

	o.client = grpcclient.NewMetricsClient(o.config.AnalyserAddress)
	if err := o.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	log.Printf("Connected to Analyser")
	return nil
}

// connectNATS establishes connection to NATS event bus for real-time Dashboard metrics.
// This is an optional connection - failure logs a warning but does not prevent startup.
func (o *Orchestrator) connectNATS() {
	if o.config.NatsURL == "" {
		log.Printf("NATS URL not configured, skipping connection")
		return
	}

	log.Printf("Connecting to NATS at: %s", o.config.NatsURL)

	publisher, err := eventbus.NewPublisher(o.config.NatsURL)
	if err != nil {
		log.Printf("Warning: failed to connect to NATS - Dashboard real-time metrics unavailable: %v", err)
		return
	}

	o.natsPublisher = publisher
	log.Printf("Connected to NATS")
}

// connectKnowledge establishes gRPC connection to Knowledge service and registers the database.
// This is an optional connection - failure logs a warning but does not prevent startup.
// Without Knowledge registration, Executor cannot fetch connection strings for autonomous actions.
func (o *Orchestrator) connectKnowledge() {
	client, err := knowledge.NewClient(o.config.KnowledgeAddress)
	if err != nil {
		log.Printf("Warning: failed to connect to Knowledge service: %v", err)
		return
	}

	o.knowledgeClient = client

	// Register database with Knowledge
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := &knowledge.DatabaseInfo{
		DatabaseID:       o.config.DatabaseID,
		ConnectionString: o.config.DBConnectionString,
		DatabaseType:     o.config.DBAdapter,
		DatabaseName:     o.config.DatabaseName,
	}

	if err := client.RegisterDatabase(ctx, info); err != nil {
		log.Printf("Warning: failed to register database with Knowledge: %v", err)
		log.Printf("Executor will not be able to deploy autonomous actions without database registration")
	}
}

// Run starts the periodic metric collection loop.
// Metrics are collected at the configured interval and sent to Analyser and NATS.
//
// The loop continues until the provided context is cancelled.
// Any errors during collection are logged but do not stop the loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting metric collection (interval: %v)", o.config.CollectionInterval)

	ticker := time.NewTicker(o.config.CollectionInterval)
	defer ticker.Stop()

	// Perform initial collection immediately
	if err := o.collectAndSend(ctx); err != nil {
		log.Printf("Error in initial collection cycle: %v", err)
	}

	// Continue periodic collection
	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutting down metric collection")
			return ctx.Err()

		case <-ticker.C:
			if err := o.collectAndSend(ctx); err != nil {
				log.Printf("Error in collection cycle: %v", err)
			}
		}
	}
}

// collectAndSend performs a single metric collection cycle:
//  1. Collect raw metrics from database
//  2. Normalize metrics to standard format
//  3. Send to Analyser via gRPC
//  4. Publish to NATS for Dashboard (if connected)
func (o *Orchestrator) collectAndSend(ctx context.Context) error {
	log.Printf("--- Collection Cycle Start ---")

	// Collect raw metrics from database
	log.Printf("Collecting metrics from database...")
	rawMetrics, err := o.adapter.CollectMetrics()
	if err != nil {
		return fmt.Errorf("metric collection failed: %w", err)
	}

	// Normalize metrics to standard format
	log.Printf("Normalising metrics...")
	normalised, err := o.normaliser.Normalise(rawMetrics)
	if err != nil {
		return fmt.Errorf("normalization failed: %w", err)
	}

	log.Printf("Metrics normalised - Health Score: %.2f, Available: %v", normalised.HealthScore, normalised.AvailableMetrics)

	// Convert to protobuf for gRPC transmission
	snapshot := o.toProtobuf(normalised)

	// Send to Analyser (required)
	log.Printf("Sending metrics to Analyser...")
	ack, err := o.client.StreamMetrics(ctx, []*pb.MetricSnapshot{snapshot})
	if err != nil {
		return fmt.Errorf("failed to send metrics to Analyser: %w", err)
	}

	log.Printf("Metrics sent successfully - Ack: %d metrics, status: %s", ack.TotalMetrics, ack.Status)

	// Publish to NATS for Dashboard (optional)
	if o.natsPublisher != nil {
		if err := o.natsPublisher.PublishMetrics(normalised); err != nil {
			log.Printf("Warning: failed to publish metrics to NATS: %v", err)
		} else {
			log.Printf("Metrics published to event bus [%s]", normalised.DatabaseID)
		}
	}

	log.Printf("--- Collection Cycle Complete ---")
	return nil
}

// Stop gracefully closes all connections and releases resources.
// This method should be called during application shutdown.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	// Close gRPC connection to Analyser
	if o.client != nil {
		if err := o.client.Close(); err != nil {
			log.Printf("Error closing Analyser connection: %v", err)
		}
	}

	// Close NATS connection
	if o.natsPublisher != nil {
		o.natsPublisher.Close()
	}

	// Close Knowledge client (handles its own gRPC connection)
	if o.knowledgeClient != nil {
		if err := o.knowledgeClient.Close(); err != nil {
			log.Printf("Error closing Knowledge client: %v", err)
		}
	}

	// Close database connection
	if o.adapter != nil {
		if err := o.adapter.Close(); err != nil {
			log.Printf("Error closing database adapter: %v", err)
		}
	}

	log.Printf("Orchestrator stopped successfully")
	return nil
}

// toProtobuf converts normalized metrics to protobuf format for gRPC transmission.
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
