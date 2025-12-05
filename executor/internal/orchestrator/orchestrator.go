package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/eventbus"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	httpserver "github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/http"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/knowledge"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
)

// Orchestrator manages the Executor service lifecycle and coordinates
// action execution, event handling, and communication with downstream services.
//
// Lifecycle:
//  1. Start() - Initializes detection handler, NATS, Knowledge, HTTP, and gRPC servers
//  2. Run() - Starts all servers and blocks until context is cancelled
//  3. Stop() - Gracefully closes all connections and resources
//
// The orchestrator implements graceful degradation:
//   - NATS failure: Actions cannot be received or published (service non-functional)
//   - Knowledge failure: Actions proceed but not registered (no deduplication or status tracking)
//   - HTTP server failure: Rollback API unavailable (but autonomous actions continue)
type Orchestrator struct {
	config *config.Config

	// Core components
	detectionHandler *handler.DetectionHandler

	// Downstream service connections
	natsPublisher   *eventbus.Publisher  // NATS publisher for action status
	natsSubscriber  *eventbus.Subscriber // NATS subscriber for detections
	knowledgeClient *knowledge.Client    // Knowledge service client

	// Servers
	httpServer   *httpserver.Server
	grpcServer   *grpc.Server
	grpcListener net.Listener
}

// NewOrchestrator creates a new Orchestrator instance with the provided configuration.
// The orchestrator is not started until Start() is called.
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

// Start initializes all service connections and prepares the orchestrator for action execution.
// This method must be called before Run().
//
// Start connects to:
//   - Knowledge service via gRPC (optional - for action registration and deduplication)
//   - NATS event bus (required - for receiving detections and publishing action status)
//   - Detection handler (required - executes actions based on detections)
//   - HTTP server (optional - provides rollback API for Dashboard)
//   - gRPC server (required - provides action status API)
//
// Returns an error if any required component fails to initialize.
func (o *Orchestrator) Start() error {
	log.Printf("Starting Executor Orchestrator...")

	// Connect to downstream services
	o.connectKnowledge() // Optional - warnings logged on failure
	if err := o.connectNATS(); err != nil {
		return fmt.Errorf("failed to connect to NATS (required): %w", err)
	}

	// Initialize detection handler
	if err := o.initializeDetectionHandler(); err != nil {
		return fmt.Errorf("failed to initialize detection handler: %w", err)
	}

	// Initialize servers
	if err := o.initializeHTTPServer(); err != nil {
		log.Printf("Warning: failed to initialize HTTP server: %v", err)
		log.Printf("Rollback API will be unavailable")
	}

	if err := o.initializeGRPCServer(); err != nil {
		return fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	log.Printf("Executor Orchestrator started successfully")
	return nil
}

// connectKnowledge establishes gRPC connection to Knowledge service for action registration and deduplication.
// This is an optional connection - failure logs a warning but does not prevent startup.
// Without Knowledge connection, actions proceed but are not tracked or deduplicated.
func (o *Orchestrator) connectKnowledge() {
	log.Printf("Connecting to Knowledge service at: %s", o.config.KnowledgeAddress)

	client, err := knowledge.NewClient(o.config.KnowledgeAddress)
	if err != nil {
		log.Printf("Warning: failed to connect to Knowledge service: %v", err)
		log.Printf("Actions will execute but not be registered or deduplicated")
		return
	}

	o.knowledgeClient = client
	log.Printf("Connected to Knowledge service")
}

// connectNATS establishes connection to NATS event bus for receiving detections and publishing action status.
// This is a REQUIRED connection - without NATS, the Executor cannot receive detections or publish status.
func (o *Orchestrator) connectNATS() error {
	log.Printf("Connecting to NATS at: %s", o.config.NatsURL)

	// Initialize publisher for action status
	publisher, err := eventbus.NewPublisher(o.config.NatsURL)
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}
	o.natsPublisher = publisher
	log.Printf("Connected to NATS publisher")

	// Note: Subscriber is initialized after detection handler is created
	// because it needs the handler instance

	return nil
}

// initializeDetectionHandler creates the detection handler that processes incoming detections.
func (o *Orchestrator) initializeDetectionHandler() error {
	log.Printf("Initializing detection handler...")

	o.detectionHandler = handler.NewDetectionHandler(o.natsPublisher, o.knowledgeClient)
	log.Printf("Detection handler initialized")

	// Now initialize NATS subscriber with the handler
	subscriber, err := eventbus.NewSubscriber(o.config.NatsURL, o.detectionHandler)
	if err != nil {
		return fmt.Errorf("failed to create NATS subscriber: %w", err)
	}

	if err := subscriber.Start(); err != nil {
		return fmt.Errorf("failed to start NATS subscriber: %w", err)
	}

	o.natsSubscriber = subscriber
	log.Printf("NATS subscriber started - listening for detections")

	return nil
}

// initializeHTTPServer creates the HTTP server for the rollback API.
// This server provides REST endpoints for Dashboard to trigger action rollbacks.
func (o *Orchestrator) initializeHTTPServer() error {
	log.Printf("Initializing HTTP server on port: %s", o.config.HTTPPort)

	o.httpServer = httpserver.NewServer(o.detectionHandler)

	log.Printf("HTTP server initialized on port %s", o.config.HTTPPort)
	return nil
}

// initializeGRPCServer creates the gRPC server for action status queries.
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

	// Register executor service
	executorServer := grpcserver.NewExecutorSever()
	pb.RegisterExecutorServiceServer(o.grpcServer, executorServer)

	log.Printf("gRPC server initialized on port %s", o.config.GRPCPort)
	return nil
}

// Run starts all servers and blocks until the context is cancelled or an error occurs.
// Actions are received from NATS, executed autonomously, and status is published back to NATS.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting servers...")

	// Start HTTP server in background (if initialized)
	httpErrChan := make(chan error, 1)
	if o.httpServer != nil {
		go func() {
			addr := ":" + o.config.HTTPPort
			log.Printf("HTTP server listening on port %s", o.config.HTTPPort)
			if err := o.httpServer.Start(addr); err != nil {
				httpErrChan <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	}

	// Start gRPC server in background
	grpcErrChan := make(chan error, 1)
	go func() {
		log.Printf("gRPC server listening on port %s", o.config.GRPCPort)
		if err := o.grpcServer.Serve(o.grpcListener); err != nil {
			grpcErrChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	log.Printf("Executor ready - listening for detections on NATS")

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
		return ctx.Err()
	case err := <-httpErrChan:
		return err
	case err := <-grpcErrChan:
		return err
	}
}

// Stop gracefully closes all connections and releases resources.
// This method should be called during application shutdown.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	// Stop HTTP server gracefully
	if o.httpServer != nil {
		if err := o.httpServer.Stop(); err != nil {
			log.Printf("Error stopping HTTP server: %v", err)
		}
	}

	// Stop gRPC server (graceful shutdown)
	if o.grpcServer != nil {
		log.Printf("Stopping gRPC server...")
		o.grpcServer.GracefulStop()
	}

	// Close NATS subscriber
	if o.natsSubscriber != nil {
		o.natsSubscriber.Close()
	}

	// Close NATS publisher
	if o.natsPublisher != nil {
		o.natsPublisher.Close()
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
