package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/orchestrator"
)

// main is the entry point for the Executor service.
//
// The Executor is responsible for:
//   - Subscribing to detection events from NATS (published by Analyser)
//   - Executing autonomous database optimization actions
//   - Publishing action status updates to NATS for Dashboard visibility
//   - Registering actions with Knowledge for deduplication and tracking
//   - Providing HTTP API for action rollback (triggered by Dashboard)
//   - Providing gRPC API for action status queries
//
// Lifecycle:
//  1. Load configuration from environment variables
//  2. Initialize orchestrator with detection handler and service connections
//  3. Start health check server (port 8082)
//  4. Start HTTP server (rollback API) and gRPC server (status API)
//  5. Listen for shutdown signals (SIGINT, SIGTERM)
//  6. Gracefully close all connections on shutdown
func main() {
	log.Printf("StartupMonkey Executor starting...")

	// Load configuration from environment variables and .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  gRPC Port: %s", cfg.GRPCPort)
	log.Printf("  HTTP Port: %s", cfg.HTTPPort)
	log.Printf("  Health Port: %s", cfg.HealthPort)
	log.Printf("  NATS URL: %s", cfg.NatsURL)
	log.Printf("  Knowledge Address: %s", cfg.KnowledgeAddress)
	log.Printf("  Auto-Execution Enabled: %v", cfg.EnableAutoExecution)
	log.Printf("  Max Concurrent Actions: %d", cfg.MaxConcurrentActions)
	log.Printf("  Action Timeout: %ds", cfg.ActionTimeout)

	// Create orchestrator to manage service lifecycle
	orch := orchestrator.NewOrchestrator(cfg)

	// Initialize all service connections and handlers
	if err := orch.Start(); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Setup graceful shutdown handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for shutdown signals (Ctrl+C, Docker stop, k8s termination)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start health check HTTP server for container orchestration
	// Exposes /health endpoint on configured port (default: 8082)
	health.StartHealthCheckServer(cfg.HealthPort)

	// Start HTTP and gRPC servers in background goroutine
	go func() {
		if err := orch.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	// Block until shutdown signal received
	<-sigChan
	log.Printf("Shutdown signal received, initiating graceful shutdown...")

	// Cancel context to stop servers
	cancel()

	// Close all connections and cleanup resources
	if err := orch.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("Executor stopped successfully")
}
