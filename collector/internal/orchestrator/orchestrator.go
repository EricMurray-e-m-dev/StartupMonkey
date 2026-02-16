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
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/system"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

// Orchestrator manages the Collector service lifecycle and coordinates
// metric collection, normalization, and distribution to downstream services.
type Orchestrator struct {
	config *config.Config

	// Database connection and metric collection
	adapter    adapter.MetricAdapter
	normaliser normaliser.Normaliser

	// Downstream service connections
	client          *grpcclient.MetricsClient
	natsPublisher   *eventbus.Publisher
	knowledgeClient *knowledge.Client
}

func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

// Start initializes all service connections and prepares the orchestrator for metric collection.
func (o *Orchestrator) Start(ctx context.Context) error {
	log.Printf("Starting Collector Orchestrator...")

	// Connect to Knowledge first - we need config from there
	if err := o.connectKnowledge(); err != nil {
		return fmt.Errorf("failed to connect to Knowledge: %w", err)
	}

	// Wait for system config from Knowledge (user must complete onboarding)
	if err := o.waitForConfig(ctx); err != nil {
		return fmt.Errorf("failed to get config from Knowledge: %w", err)
	}

	// Now we have database config, connect to database
	if err := o.connectDatabase(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize downstream services
	if err := o.connectAnalyser(); err != nil {
		return fmt.Errorf("failed to connect to Analyser: %w", err)
	}

	o.connectNATS()

	// Register database with Knowledge
	o.registerDatabase()

	log.Printf("Collector Orchestrator started successfully")
	return nil
}

// connectKnowledge establishes gRPC connection to Knowledge service.
func (o *Orchestrator) connectKnowledge() error {
	client, err := knowledge.NewClient(o.config.KnowledgeAddress)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	o.knowledgeClient = client
	log.Printf("Connected to Knowledge service")
	return nil
}

// waitForConfig polls Knowledge for system configuration.
// Blocks until configuration is available (onboarding complete).
func (o *Orchestrator) waitForConfig(ctx context.Context) error {
	log.Printf("Waiting for system configuration from Knowledge...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			config, err := o.knowledgeClient.GetSystemConfig(ctx)
			if err != nil {
				log.Printf("Failed to get config: %v (retrying...)", err)
				continue
			}

			if config == nil || config.Database == nil || config.Database.ConnectionString == "" {
				log.Printf("Awaiting configuration... (complete onboarding in Dashboard)")
				continue
			}

			if !config.OnboardingComplete {
				log.Printf("Awaiting configuration... (onboarding not complete)")
				continue
			}

			// Config received - apply to local config
			log.Printf("Configuration received from Knowledge")
			o.config.SetDatabaseConfig(
				config.Database.ConnectionString,
				config.Database.Type,
				config.Database.Id,
				config.Database.Name,
			)

			if err := o.config.ValidateFull(); err != nil {
				log.Printf("Invalid config from Knowledge: %v (retrying...)", err)
				continue
			}

			log.Printf("  Database ID: %s", o.config.DatabaseID)
			log.Printf("  Database Type: %s", o.config.DBAdapter)
			log.Printf("  Database Name: %s", o.config.DatabaseName)

			return nil
		}
	}
}

// connectDatabase establishes connection to the target database.
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
	health.SetUnavailableFeatures(o.adapter.GetUnavailableFeatures())

	log.Printf("Database connected and healthy")
	return nil
}

// connectAnalyser establishes gRPC connection to the Analyser service.
func (o *Orchestrator) connectAnalyser() error {
	log.Printf("Connecting to Analyser at: %s", o.config.AnalyserAddress)

	o.client = grpcclient.NewMetricsClient(o.config.AnalyserAddress)
	if err := o.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	log.Printf("Connected to Analyser")
	return nil
}

// connectNATS establishes connection to NATS event bus.
func (o *Orchestrator) connectNATS() {
	if o.config.NatsURL == "" {
		log.Printf("NATS URL not configured, skipping connection")
		return
	}

	log.Printf("Connecting to NATS at: %s", o.config.NatsURL)

	publisher, err := eventbus.NewPublisher(o.config.NatsURL)
	if err != nil {
		log.Printf("Warning: failed to connect to NATS: %v", err)
		return
	}

	o.natsPublisher = publisher
	log.Printf("Connected to NATS")
}

// registerDatabase registers the database with Knowledge service.
func (o *Orchestrator) registerDatabase() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := &knowledge.DatabaseInfo{
		DatabaseID:       o.config.DatabaseID,
		ConnectionString: o.config.DBConnectionString,
		DatabaseType:     o.config.DBAdapter,
		DatabaseName:     o.config.DatabaseName,
	}

	if err := o.knowledgeClient.RegisterDatabase(ctx, info); err != nil {
		log.Printf("Warning: failed to register database with Knowledge: %v", err)
	}
}

// Run starts the periodic metric collection loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting metric collection (interval: %v)", o.config.CollectionInterval)

	ticker := time.NewTicker(o.config.CollectionInterval)
	defer ticker.Stop()

	// Perform initial collection immediately
	if err := o.collectAndSend(ctx); err != nil {
		log.Printf("Error in initial collection cycle: %v", err)
	}

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

// collectAndSend performs a single metric collection cycle.
func (o *Orchestrator) collectAndSend(ctx context.Context) error {
	log.Printf("--- Collection Cycle Start ---")

	log.Printf("Collecting metrics from database...")
	rawMetrics, err := o.adapter.CollectMetrics()
	if err != nil {
		return fmt.Errorf("metric collection failed: %w", err)
	}

	// Collect system metrics
	sysMetrics, err := system.Collect()
	if err != nil {
		log.Printf("Warning: failed to collect system metrics: %v", err)
	} else {
		for k, v := range sysMetrics.ToExtendedMetrics() {
			rawMetrics.ExtendedMetrics[k] = v
		}
	}

	log.Printf("Normalising metrics...")
	normalised, err := o.normaliser.Normalise(rawMetrics)
	if err != nil {
		return fmt.Errorf("normalization failed: %w", err)
	}

	log.Printf("Metrics normalised - Health Score: %.2f, Available: %v", normalised.HealthScore, normalised.AvailableMetrics)

	snapshot := o.toProtobuf(normalised)

	log.Printf("Sending metrics to Analyser...")
	ack, err := o.client.StreamMetrics(ctx, []*pb.MetricSnapshot{snapshot})
	if err != nil {
		return fmt.Errorf("failed to send metrics to Analyser: %w", err)
	}

	log.Printf("Metrics sent successfully - Ack: %d metrics, status: %s", ack.TotalMetrics, ack.Status)

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

// Stop gracefully closes all connections.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	if o.client != nil {
		if err := o.client.Close(); err != nil {
			log.Printf("Error closing Analyser connection: %v", err)
		}
	}

	if o.natsPublisher != nil {
		o.natsPublisher.Close()
	}

	if o.knowledgeClient != nil {
		if err := o.knowledgeClient.Close(); err != nil {
			log.Printf("Error closing Knowledge client: %v", err)
		}
	}

	if o.adapter != nil {
		if err := o.adapter.Close(); err != nil {
			log.Printf("Error closing database adapter: %v", err)
		}
	}

	log.Printf("Orchestrator stopped successfully")
	return nil
}

// toProtobuf converts normalized metrics to protobuf format.
func (o *Orchestrator) toProtobuf(n *normaliser.NormalisedMetrics) *pb.MetricSnapshot {
	snapshot := &pb.MetricSnapshot{
		DatabaseId:   n.DatabaseID,
		DatabaseType: n.DatabaseType,
		Timestamp:    n.Timestamp,

		HealthScore:      n.HealthScore,
		ConnectionHealth: n.ConnectionHealth,
		QueryHealth:      n.QueryHealth,
		StorageHealth:    n.StorageHealth,
		CacheHealth:      n.CacheHealth,

		AvailableMetrics: n.AvailableMetrics,
		MetricDeltas:     n.MetricDeltas,
		TimeDeltaSeconds: &n.TimeDeltaSeconds,

		ExtendedMetrics: n.ExtendedMetrics,
		Labels:          n.Labels,

		Measurements: &pb.Measurements{
			ActiveConnections:  n.Measurements.ActiveConnections,
			IdleConnections:    n.Measurements.IdleConnections,
			MaxConnections:     n.Measurements.MaxConnections,
			WaitingConnections: n.Measurements.WaitingConnections,

			AvgQueryLatencyMs: n.Measurements.AvgQueryLatencyMs,
			P50QueryLatencyMs: n.Measurements.P50QueryLatencyMs,
			P95QueryLatencyMs: n.Measurements.P95QueryLatencyMs,
			P99QueryLatencyMs: n.Measurements.P99QueryLatencyMs,
			SlowQueryCount:    n.Measurements.SlowQueryCount,
			SequentialScans:   n.Measurements.SequentialScans,

			UsedStorageBytes:  n.Measurements.UsedStorageBytes,
			TotalStorageBytes: n.Measurements.TotalStorageBytes,
			FreeStorageBytes:  n.Measurements.FreeStorageBytes,

			CacheHitRate:   n.Measurements.CacheHitRate,
			CacheHitCount:  n.Measurements.CacheHitCount,
			CacheMissCount: n.Measurements.CacheMissCount,
		},
	}

	return snapshot
}
