package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/config"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
)

// Orchestrator manages the Knowledge service lifecycle and coordinates
// state management, Redis operations, and gRPC/HTTP server management.
//
// Lifecycle:
//  1. Start() - Initializes Redis client, gRPC server, and health check server
//  2. Run() - Starts all servers and blocks until context is cancelled
//  3. Stop() - Gracefully closes all connections and resources
//
// The Knowledge service acts as the central state store for the entire system:
//   - Stores database connection information (for Executor actions)
//   - Tracks detection state (for Analyser deduplication)
//   - Manages action status (for Dashboard visibility)
//   - Provides health monitoring (Redis connectivity)
type Orchestrator struct {
	config *config.Config

	// Core components
	redisClient *redis.Client

	// Servers
	healthServer *health.HealthServer
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

// Start initializes all service connections and prepares the orchestrator for operation.
// This method must be called before Run().
//
// Start connects to:
//   - Redis (required - central state store)
//   - gRPC server (required - provides Knowledge API)
//   - Health check server (optional - monitoring endpoint)
//
// Returns an error if any required component fails to initialize.
func (o *Orchestrator) Start() error {
	log.Printf("Starting Knowledge Orchestrator...")

	// Connect to Redis (required)
	if err := o.connectRedis(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize servers
	if err := o.initializeGRPCServer(); err != nil {
		return fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	if err := o.initializeHealthServer(); err != nil {
		log.Printf("Warning: failed to initialize health server: %v", err)
		log.Printf("Health check endpoint will be unavailable")
	}

	log.Printf("Knowledge Orchestrator started successfully")
	return nil
}

// connectRedis establishes connection to Redis for state storage.
// This is a REQUIRED connection - without Redis, Knowledge cannot function.
func (o *Orchestrator) connectRedis() error {
	log.Printf("Connecting to Redis at: %s (DB: %d)", o.config.RedisAddr, o.config.RedisDB)

	client, err := redis.NewClient(o.config.RedisAddr, o.config.RedisPassword, o.config.RedisDB)
	if err != nil {
		return fmt.Errorf("failed to create Redis client: %w", err)
	}

	o.redisClient = client
	log.Printf("Connected to Redis")
	return nil
}

// initializeGRPCServer creates the gRPC server for the Knowledge API.
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

	// Register Knowledge service with Redis client
	knowledgeServer := grpcserver.NewKnowledgeServer(o.redisClient)
	pb.RegisterKnowledgeServiceServer(o.grpcServer, knowledgeServer)

	log.Printf("gRPC server initialized on port %s", o.config.GRPCPort)
	return nil
}

// initializeHealthServer creates the HTTP health check server.
func (o *Orchestrator) initializeHealthServer() error {
	log.Printf("Initializing health check server on port: %s", o.config.HealthPort)

	o.healthServer = health.NewHealthServer(o.redisClient)

	log.Printf("Health check server initialized on port %s", o.config.HealthPort)
	return nil
}

// Run starts all servers and blocks until the context is cancelled or an error occurs.
// Knowledge API is available via gRPC, health checks via HTTP.
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("Starting servers...")

	// Start health check server in background (if initialized)
	healthErrChan := make(chan error, 1)
	if o.healthServer != nil {
		go func() {
			addr := ":" + o.config.HealthPort
			log.Printf("Health check server listening on port %s", o.config.HealthPort)
			if err := o.healthServer.Start(addr); err != nil {
				healthErrChan <- fmt.Errorf("health check server error: %w", err)
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

	log.Printf("Knowledge service ready - central state store active")

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
		return ctx.Err()
	case err := <-healthErrChan:
		return err
	case err := <-grpcErrChan:
		return err
	}
}

// Stop gracefully closes all connections and releases resources.
// This method should be called during application shutdown.
func (o *Orchestrator) Stop() error {
	log.Printf("Stopping Orchestrator...")

	// Stop gRPC server (graceful shutdown)
	if o.grpcServer != nil {
		log.Printf("Stopping gRPC server...")
		o.grpcServer.GracefulStop()
	}

	// Stop health check server
	if o.healthServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := o.healthServer.Shutdown(ctx); err != nil {
			log.Printf("Error stopping health server: %v", err)
		}
	}

	// Close Redis connection
	if o.redisClient != nil {
		if err := o.redisClient.Close(); err != nil {
			log.Printf("Error closing Redis client: %v", err)
		}
	}

	log.Printf("Orchestrator stopped successfully")
	return nil
}
