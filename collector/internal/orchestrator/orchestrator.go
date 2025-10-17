package orchestrator

import (
	"context"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type Orchestrator struct {
	config  *config.Config
	adapter adapter.MetricAdapter
	client  *grpcclient.MetricsClient
}

func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

func (o *Orchestrator) Start() error {
	log.Printf("Starting Collector Orchestrator...")

	var err error
	o.adapter, err = adapter.NewAdapter(o.config.DBAdapter, o.config.DBConnectionString)
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

	o.client = grpcclient.NewMetricsClient(o.config.AnalyserAddress)
	if err := o.client.Connect(); err != nil {
		return err
	}
	log.Printf("Connected to Analyser at %s", o.config.AnalyserAddress)

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

	log.Printf("Metrics collected: %d active connections, %.2f%% cache hit rate", rawMetrics.ActiveConnections, rawMetrics.CacheHitRate*100)

	snapshot := &pb.MetricsSnapshot{
		DatabaseId:        o.config.DatabaseID,
		Timestamp:         rawMetrics.Timestamp,
		DatabaseType:      rawMetrics.DatabaseType,
		CpuPercent:        rawMetrics.CPUPercent,
		MemoryPercent:     rawMetrics.MemoryPercent,
		ActiveConnections: rawMetrics.ActiveConnections,
		IdleConnections:   rawMetrics.IdleConnections,
		MaxConnections:    rawMetrics.MaxConnections,
		CacheHitRate:      rawMetrics.CacheHitRate,
		ExtendedMetrics:   rawMetrics.ExtendedMetrics,
	}

	log.Printf("Sending metrics to analyser....")
	ack, err := o.client.StreamMetrics(ctx, []*pb.MetricsSnapshot{snapshot})
	if err != nil {
		return err
	}

	log.Printf("Metrics sent successfully. Ack: %d metrics, status: %s", ack.TotalMetrics, ack.Status)
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

	if o.adapter != nil {
		if err := o.adapter.Close(); err != nil {
			log.Printf("Error closing database adapter: %v", err)
		}
	}

	log.Printf("Orchestator stopped")
	return nil

}
