package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/eventbus"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/verification"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Orchestrator manages the Analyser service lifecycle and coordinates
// metric analysis, detection, and event publishing.
//
// Lifecycle:
//  1. Start() - Initializes detection engine, NATS, Knowledge, and gRPC server
//  2. Run() - Starts gRPC server to receive metrics from Collector
//  3. Stop() - Gracefully closes all connections and resources
//
// The orchestrator implements graceful degradation:
//   - NATS failure: Detections registered with Knowledge but not published (Executor unavailable)
//   - Knowledge failure: Detections published to NATS but not deduplicated (may cause duplicate actions)
type Orchestrator struct {
	config *config.Config

	// Detection engine and registered detectors
	engine *engine.Engine

	// Downstream service connections
	publisher       *eventbus.Publisher        // NATS publisher for detections
	subscriber      *eventbus.Subscriber       // NATS subscriber for action completions
	knowledgeClient *knowledge.KnowledgeClient // Knowledge service client

	// gRPC server
	grpcServer   *grpc.Server
	grpcListener net.Listener

	// Verification tracker for auto rollback (temporary)
	verificationTracker *verification.Tracker
}

// NewOrchestrator creates a new Orchestrator instance with the provided configuration.
// The orchestrator is not started until Start() is called.
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

func (o *Orchestrator) Start() error {
	log.Printf("Starting Analyser Orchestrator...")

	// Connect to Knowledge first - we may need thresholds from there
	o.connectKnowledge()

	// Try to fetch thresholds from Knowledge (overrides defaults)
	o.fetchThresholdsFromKnowledge()

	// Initialize detection engine with configured thresholds
	if err := o.initializeEngine(); err != nil {
		return fmt.Errorf("failed to initialize detection engine: %w", err)
	}

	// Verification setup
	o.initializeVerificationTracker()

	// Connect to NATS
	o.connectNATS()

	// Initialize gRPC server to receive metrics
	if err := o.initializeGRPCServer(); err != nil {
		return fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	log.Printf("Analyser Orchestrator started successfully")
	return nil
}

// fetchThresholdsFromKnowledge attempts to load detection thresholds from Knowledge service.
// If successful, overrides the default/env var thresholds in config.
// If Knowledge is unavailable or unconfigured, falls back to existing thresholds.
func (o *Orchestrator) fetchThresholdsFromKnowledge() {
	if o.knowledgeClient == nil {
		log.Printf("Knowledge client unavailable - using default thresholds")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := o.knowledgeClient.GetSystemConfig(ctx)
	if err != nil {
		log.Printf("Failed to fetch config from Knowledge: %v - using default thresholds", err)
		return
	}

	if config == nil || config.Thresholds == nil {
		log.Printf("No thresholds configured in Knowledge - using default thresholds")
		return
	}

	log.Printf("Applying thresholds from Knowledge service...")

	thresholds := config.Thresholds

	// Only override if values are set (non-zero)
	if thresholds.ConnectionPoolCritical > 0 {
		o.config.Thresholds.ConnectionPoolCritical = thresholds.ConnectionPoolCritical
		log.Printf("  - Connection Pool Critical: %.2f", thresholds.ConnectionPoolCritical)
	}

	if thresholds.SequentialScanThreshold > 0 {
		o.config.Thresholds.SequentialScanThreshold = int32(thresholds.SequentialScanThreshold)
		log.Printf("  - Sequential Scan Threshold: %d", thresholds.SequentialScanThreshold)
	}

	if thresholds.SequentialScanDelta > 0 {
		o.config.Thresholds.SequentialScanDeltaThreshold = thresholds.SequentialScanDelta
		log.Printf("  - Sequential Scan Delta: %.1f", thresholds.SequentialScanDelta)
	}

	if thresholds.P95LatencyMs > 0 {
		o.config.Thresholds.P95LatencyThresholdMs = thresholds.P95LatencyMs
		log.Printf("  - P95 Latency: %.0fms", thresholds.P95LatencyMs)
	}

	if thresholds.CacheHitRateThreshold > 0 {
		o.config.Thresholds.CacheHitRateThreshold = thresholds.CacheHitRateThreshold
		log.Printf("  - Cache Hit Rate: %.2f", thresholds.CacheHitRateThreshold)
	}

	log.Printf("Thresholds loaded from Knowledge")
}

// initializeEngine creates the detection engine and registers all detectors with configured thresholds.
func (o *Orchestrator) initializeEngine() error {
	log.Printf("Initializing detection engine...")

	o.engine = engine.NewEngine()

	if !o.config.EnableAllDetectors {
		log.Printf("Warning: Not all detectors enabled (ENABLE_ALL_DETECTORS=false)")
		return nil
	}

	// Register detectors with configured thresholds
	o.registerDetectors()

	detectorNames := o.engine.GetRegisteredDetectors()
	log.Printf("Detection engine initialized with %d detectors: %v", len(detectorNames), detectorNames)
	return nil
}

// registerDetectors registers all available detectors with the engine and applies configured thresholds.
// registerDetectors registers all available detectors with the engine and applies configured thresholds.
func (o *Orchestrator) registerDetectors() {
	log.Printf("Registering detectors with configured thresholds...")

	// Connection Pool Detector
	connPoolDetector := detector.NewConnectionPoolDetection()
	connPoolDetector.SetThreshold(o.config.Thresholds.ConnectionPoolCritical)
	o.engine.RegisterDetector(connPoolDetector)
	log.Printf("  - Connection Pool: threshold=%.2f (%.0f%%)",
		o.config.Thresholds.ConnectionPoolCritical,
		o.config.Thresholds.ConnectionPoolCritical*100)

	// Missing Index Detector
	missingIndexDetector := detector.NewMissingIndexDetector()
	missingIndexDetector.SetThreshold(o.config.Thresholds.SequentialScanThreshold)
	missingIndexDetector.SetDeltaThreshold(o.config.Thresholds.SequentialScanDeltaThreshold)
	o.engine.RegisterDetector(missingIndexDetector)
	log.Printf("  - Missing Index: seq_scan_threshold=%d, delta_threshold=%.1f",
		o.config.Thresholds.SequentialScanThreshold,
		o.config.Thresholds.SequentialScanDeltaThreshold)

	// High Latency Detector
	latencyDetector := detector.NewHighLatencyDetector()
	latencyDetector.SetThreshold(o.config.Thresholds.P95LatencyThresholdMs)
	o.engine.RegisterDetector(latencyDetector)
	log.Printf("  - High Latency: p95_threshold=%.0fms",
		o.config.Thresholds.P95LatencyThresholdMs)

	// Cache Miss Detector
	cacheMissDetector := detector.NewCacheMissDetector()
	cacheMissDetector.SetThreshold(o.config.Thresholds.CacheHitRateThreshold)
	o.engine.RegisterDetector(cacheMissDetector)
	log.Printf("  - Cache Miss: hit_rate_threshold=%.2f (%.0f%%)",
		o.config.Thresholds.CacheHitRateThreshold,
		o.config.Thresholds.CacheHitRateThreshold*100)

	// Table Bloat Detector
	tableBloatDetector := detector.NewTableBloatDetector()
	tableBloatDetector.SetThreshold(o.config.Thresholds.TableBloatThreshold)
	o.engine.RegisterDetector(tableBloatDetector)
	log.Printf("  - Table Bloat: threshold=%.0f%%", o.config.Thresholds.TableBloatThreshold*100)

	// Long Running Query Detector
	longQueryDetector := detector.NewLongRunningQueryDetector()
	longQueryDetector.SetThreshold(o.config.Thresholds.LongRunningQueryThresholdSecs)
	o.engine.RegisterDetector(longQueryDetector)
	log.Printf("  - Long Running Query: threshold=%.0fs", o.config.Thresholds.LongRunningQueryThresholdSecs)

	// Idle Transaction Detector
	idleTxnDetector := detector.NewIdleTransactionDetector()
	idleTxnDetector.SetThreshold(o.config.Thresholds.IdleTransactionThresholdSecs)
	o.engine.RegisterDetector(idleTxnDetector)
	log.Printf("  - Idle Transaction: threshold=%.0fs", o.config.Thresholds.IdleTransactionThresholdSecs)
}

// initializeVerificationTracker creates the verification tracker for autonomous rollback.
// After an action is executed, the tracker monitors subsequent metrics to verify the action improved performance.
//
// The tracker uses a configurable number of verification cycles (currently 3) to determine success:
//   - If metrics improve or stabilize: detection marked as resolved in Knowledge layer
//   - If metrics degrade: rollback request published to NATS for Executor to revert the action
//
// Requires both NATS publisher (for rollback requests) and Knowledge client (for resolution tracking)
// to be available for full functionality. Partial functionality if either is unavailable.
func (o *Orchestrator) initializeVerificationTracker() {
	log.Printf("Initializing verification tracker...")

	o.verificationTracker = verification.NewTracker(
		3, // verification cycles

		// Rollback callback
		func(request *verification.RollbackRequest) {
			if o.publisher != nil {
				log.Printf("Verification failed - requesting rollback for action %s", request.ActionID)
				if err := o.publisher.PublishRollbackRequest(request); err != nil {
					log.Printf("Failed to publish rollback request: %v", err)
				}
			}
		},

		// Verified callback
		func(detectionID string) {
			if o.knowledgeClient != nil {
				log.Printf("Action verified - marking detection %s as resolved", detectionID)
				ctx := context.Background()
				if err := o.knowledgeClient.MarkDetectionResolved(ctx, detectionID, "verified_by_metrics"); err != nil {
					log.Printf("Failed to mark detection resolved: %v", err)
				}
			}
		},
	)

	log.Printf("Verification tracker initialized (3 cycle verification)")
}

// connectKnowledge establishes gRPC connection to Knowledge service for detection deduplication.
// This is an optional connection - failure logs a warning but does not prevent startup.
// Without Knowledge connection, duplicate detections may be published to NATS.
func (o *Orchestrator) connectKnowledge() {
	log.Printf("Connecting to Knowledge service at: %s", o.config.KnowledgeAddress)

	client, err := knowledge.NewKnowledgeClient(o.config.KnowledgeAddress)
	if err != nil {
		log.Printf("Warning: failed to connect to Knowledge service: %v", err)
		log.Printf("Detection deduplication unavailable - duplicate actions may be triggered")
		return
	}

	o.knowledgeClient = client
	log.Printf("Connected to Knowledge service")
}

// connectNATS establishes connection to NATS event bus for publishing detections and subscribing to action results.
// This is an optional connection - failure logs a warning but does not prevent startup.
// Without NATS connection, detections cannot be published to Executor (no autonomous actions).
func (o *Orchestrator) connectNATS() {
	if o.config.NatsURL == "" {
		log.Printf("NATS URL not configured, skipping connection")
		return
	}

	log.Printf("Connecting to NATS at: %s", o.config.NatsURL)

	// Initialize publisher
	publisher, err := eventbus.NewPublisher(o.config.NatsURL)
	if err != nil {
		log.Printf("Warning: failed to connect NATS publisher: %v", err)
		log.Printf("Detections will not be published - Executor unavailable")
	} else {
		o.publisher = publisher
		log.Printf("Connected to NATS publisher")
	}

	// Initialize subscriber for action completion events
	if o.knowledgeClient != nil {
		subscriber, err := eventbus.NewSubscriber(o.config.NatsURL, o.knowledgeClient, o.verificationTracker)
		if err != nil {
			log.Printf("Warning: failed to create NATS subscriber: %v", err)
			log.Printf("Action completion tracking unavailable")
		} else {
			o.subscriber = subscriber
			if err := subscriber.Start(); err != nil {
				log.Printf("Warning: failed to start NATS subscriber: %v", err)
			} else {
				log.Printf("Connected to NATS subscriber")
			}
		}
	} else {
		log.Printf("Skipping NATS subscriber (Knowledge client unavailable)")
	}
}

// initializeGRPCServer creates and configures the gRPC server to receive metrics from Collector.
func (o *Orchestrator) initializeGRPCServer() error {
	log.Printf("Initializing gRPC server on port: %s", o.config.GRPCPort)

	// Create TCP listener
	listener, err := net.Listen("tcp", ":"+o.config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", o.config.GRPCPort, err)
	}
	o.grpcListener = listener

	// Create gRPC server
	o.grpcServer = grpc.NewServer()

	// Register metrics service with detection engine, publisher, and knowledge client
	metricsServer := grpcserver.NewMetricsServer(o.engine, o.publisher, o.knowledgeClient, o.verificationTracker)
	pb.RegisterMetricsServiceServer(o.grpcServer, metricsServer)

	// Enable gRPC reflection for debugging (grpcurl, etc.)
	reflection.Register(o.grpcServer)

	log.Printf("gRPC server initialized on port %s", o.config.GRPCPort)
	return nil
}

// Run starts the gRPC server and blocks until the context is cancelled or an error occurs.
// Metrics are received from Collector, analyzed by the detection engine, and detections are published to NATS.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting gRPC server on port %s...", o.config.GRPCPort)

	// Start gRPC server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := o.grpcServer.Serve(o.grpcListener); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	log.Printf("Analyser ready - listening for metrics from Collector")

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Stop gracefully closes all connections and releases resources.
// This method should be called during application shutdown.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	// Stop gRPC server (graceful shutdown with timeout)
	if o.grpcServer != nil {
		log.Printf("Stopping gRPC server...")
		o.grpcServer.GracefulStop()
	}

	// Close NATS subscriber
	if o.subscriber != nil {
		o.subscriber.Close()
	}

	// Close NATS publisher
	if o.publisher != nil {
		o.publisher.Close()
	}

	// Close Knowledge client
	if o.knowledgeClient != nil {
		if err := o.knowledgeClient.Close(); err != nil {
			log.Printf("Error closing Knowledge client: %v", err)
		}
	}

	log.Printf("Orchestrator stopped successfully")
	return nil
}
