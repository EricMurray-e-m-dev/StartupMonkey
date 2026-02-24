// Package orchestrator manages the Collector service lifecycle and coordinates
// metric collection from multiple databases.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/eventbus"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/system"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

// AdapterEntry holds an adapter and its associated components for a single database.
type AdapterEntry struct {
	Adapter    adapter.MetricAdapter
	Normaliser normaliser.Normaliser
	DatabaseID string
	DBType     string
	DBName     string
	ConnString string
}

// Orchestrator manages the Collector service lifecycle and coordinates
// metric collection, normalization, and distribution to downstream services.
type Orchestrator struct {
	config *config.Config

	// Multi-database support
	adapters   map[string]*AdapterEntry
	adaptersMu sync.RWMutex

	// Downstream service connections
	client          *grpcclient.MetricsClient
	natsPublisher   *eventbus.Publisher
	knowledgeClient *knowledge.Client
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config:   cfg,
		adapters: make(map[string]*AdapterEntry),
	}
}

// Start initializes all service connections and prepares the orchestrator for metric collection.
func (o *Orchestrator) Start(ctx context.Context) error {
	log.Printf("Starting Collector Orchestrator...")

	// Connect to Knowledge first
	if err := o.connectKnowledge(); err != nil {
		return fmt.Errorf("failed to connect to Knowledge: %w", err)
	}

	// Wait for at least one enabled database
	if err := o.waitForDatabases(ctx); err != nil {
		return fmt.Errorf("failed to get databases from Knowledge: %w", err)
	}

	// Initialize downstream services
	if err := o.connectAnalyser(); err != nil {
		return fmt.Errorf("failed to connect to Analyser: %w", err)
	}

	o.connectNATS()

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

// waitForDatabases polls Knowledge until at least one enabled database exists.
func (o *Orchestrator) waitForDatabases(ctx context.Context) error {
	log.Printf("Waiting for enabled databases from Knowledge...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := o.syncDatabases(ctx); err != nil {
				log.Printf("Failed to sync databases: %v (retrying...)", err)
				continue
			}

			o.adaptersMu.RLock()
			count := len(o.adapters)
			o.adaptersMu.RUnlock()

			if count == 0 {
				log.Printf("No enabled databases found (complete onboarding in Dashboard)")
				continue
			}

			log.Printf("Found %d enabled database(s)", count)
			return nil
		}
	}
}

// syncDatabases synchronizes adapters with the databases registered in Knowledge.
// Adds new databases, removes unregistered/disabled ones.
func (o *Orchestrator) syncDatabases(ctx context.Context) error {
	databases, err := o.knowledgeClient.ListDatabases(ctx, true) // enabled_only=true
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	// Build set of current database IDs from Knowledge
	knownIDs := make(map[string]bool)
	for _, db := range databases {
		knownIDs[db.DatabaseId] = true
	}

	o.adaptersMu.Lock()
	defer o.adaptersMu.Unlock()

	// Remove adapters for databases no longer in Knowledge or disabled
	for id, entry := range o.adapters {
		if !knownIDs[id] {
			log.Printf("Removing adapter for database: %s (no longer enabled)", id)
			if err := entry.Adapter.Close(); err != nil {
				log.Printf("Error closing adapter for %s: %v", id, err)
			}
			delete(o.adapters, id)
		}
	}

	// Add adapters for new databases
	for _, db := range databases {
		if _, exists := o.adapters[db.DatabaseId]; exists {
			continue // Already have adapter
		}

		log.Printf("Adding adapter for database: %s (type: %s)", db.DatabaseId, db.DatabaseType)

		entry, err := o.createAdapterEntry(db)
		if err != nil {
			log.Printf("Failed to create adapter for %s: %v", db.DatabaseId, err)
			continue
		}

		o.adapters[db.DatabaseId] = entry
		log.Printf("Database connected: %s (%s)", db.DatabaseId, db.DatabaseName)
	}

	return nil
}

// createAdapterEntry creates a new adapter entry for a database.
func (o *Orchestrator) createAdapterEntry(db *pb.RegisteredDatabase) (*AdapterEntry, error) {
	adpt, err := adapter.NewAdapter(db.DatabaseType, db.ConnectionString, db.DatabaseId)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := adpt.Connect(); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	if err := adpt.HealthCheck(); err != nil {
		adpt.Close()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	return &AdapterEntry{
		Adapter:    adpt,
		Normaliser: normaliser.NewNormaliser(db.DatabaseType),
		DatabaseID: db.DatabaseId,
		DBType:     db.DatabaseType,
		DBName:     db.DatabaseName,
		ConnString: db.ConnectionString,
	}, nil
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

// Run starts the periodic metric collection loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting metric collection (interval: %v, sync: %v)",
		o.config.CollectionInterval, o.config.SyncInterval)

	collectionTicker := time.NewTicker(o.config.CollectionInterval)
	defer collectionTicker.Stop()

	syncTicker := time.NewTicker(o.config.SyncInterval)
	defer syncTicker.Stop()

	// Perform initial collection immediately
	o.collectFromAllDatabases(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutting down metric collection")
			return ctx.Err()

		case <-collectionTicker.C:
			o.collectFromAllDatabases(ctx)

		case <-syncTicker.C:
			if err := o.syncDatabases(ctx); err != nil {
				log.Printf("Error syncing databases: %v", err)
			}
		}
	}
}

// collectFromAllDatabases collects metrics from all connected databases.
func (o *Orchestrator) collectFromAllDatabases(ctx context.Context) {
	o.adaptersMu.RLock()
	entries := make([]*AdapterEntry, 0, len(o.adapters))
	for _, entry := range o.adapters {
		entries = append(entries, entry)
	}
	o.adaptersMu.RUnlock()

	if len(entries) == 0 {
		log.Printf("No databases to collect from")
		return
	}

	log.Printf("--- Collection Cycle Start (%d databases) ---", len(entries))

	// Collect system metrics once (shared across all databases)
	var sysMetrics *system.Metrics
	var sysErr error
	sysMetrics, sysErr = system.Collect()
	if sysErr != nil {
		log.Printf("Warning: failed to collect system metrics: %v", sysErr)
	}

	for _, entry := range entries {
		if err := o.collectAndSend(ctx, entry, sysMetrics); err != nil {
			log.Printf("Error collecting from %s: %v", entry.DatabaseID, err)
			// Update health status in Knowledge
			o.updateDatabaseHealth(ctx, entry.DatabaseID, "degraded", 0.5)
		} else {
			o.updateDatabaseHealth(ctx, entry.DatabaseID, "healthy", 1.0)
		}
	}

	log.Printf("--- Collection Cycle Complete ---")
}

// collectAndSend performs a single metric collection cycle for one database.
func (o *Orchestrator) collectAndSend(ctx context.Context, entry *AdapterEntry, sysMetrics *system.Metrics) error {
	log.Printf("Collecting metrics from: %s", entry.DatabaseID)

	rawMetrics, err := entry.Adapter.CollectMetrics()
	if err != nil {
		return fmt.Errorf("metric collection failed: %w", err)
	}

	// Add system metrics if available
	if sysMetrics != nil {
		for k, v := range sysMetrics.ToExtendedMetrics() {
			rawMetrics.ExtendedMetrics[k] = v
		}
	}

	normalised, err := entry.Normaliser.Normalise(rawMetrics)
	if err != nil {
		return fmt.Errorf("normalization failed: %w", err)
	}

	snapshot := o.toProtobuf(normalised)

	ack, err := o.client.StreamMetrics(ctx, []*pb.MetricSnapshot{snapshot})
	if err != nil {
		return fmt.Errorf("failed to send metrics to Analyser: %w", err)
	}

	log.Printf("  %s: Health=%.2f, Ack=%d metrics", entry.DatabaseID, normalised.HealthScore, ack.TotalMetrics)

	if o.natsPublisher != nil {
		if err := o.natsPublisher.PublishMetrics(normalised); err != nil {
			log.Printf("Warning: failed to publish metrics to NATS: %v", err)
		}
	}

	return nil
}

// updateDatabaseHealth updates the health status in Knowledge.
func (o *Orchestrator) updateDatabaseHealth(ctx context.Context, dbID, status string, score float64) {
	if err := o.knowledgeClient.UpdateDatabaseHealth(ctx, dbID, status, score); err != nil {
		log.Printf("Warning: failed to update health for %s: %v", dbID, err)
	}
}

// Stop gracefully closes all connections.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	// Close all database adapters
	o.adaptersMu.Lock()
	for id, entry := range o.adapters {
		if err := entry.Adapter.Close(); err != nil {
			log.Printf("Error closing adapter %s: %v", id, err)
		}
	}
	o.adapters = make(map[string]*AdapterEntry)
	o.adaptersMu.Unlock()

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
