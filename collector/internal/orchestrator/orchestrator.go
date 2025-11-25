package orchestrator

import (
	"context"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/eventbus"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type Orchestrator struct {
	config        *config.Config
	adapter       adapter.MetricAdapter
	normaliser    normaliser.Normaliser
	client        *grpcclient.MetricsClient
	natsPublisher *eventbus.Publisher
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
