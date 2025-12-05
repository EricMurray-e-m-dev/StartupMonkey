package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/orchestrator"
)

// main is the entry point for the Collector service.
//
// The Collector is responsible for:
//   - Connecting to the target database
//   - Collecting performance metrics at regular intervals
//   - Normalizing metrics to a standard format
//   - Streaming metrics to the Analyser service
//   - Publishing metrics to NATS for Dashboard real-time display
//   - Registering the database with Knowledge for autonomous actions
//
// Lifecycle:
//  1. Load configuration from environment variables
//  2. Initialize orchestrator with database and service connections
//  3. Start health check server (port 8080)
//  4. Begin metric collection loop
//  5. Listen for shutdown signals (SIGINT, SIGTERM)
//  6. Gracefully close all connections on shutdown
func main() {
	log.Printf("StartupMonkey Collector starting...")

	// Load configuration from environment variables and .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  Database ID: %s", cfg.DatabaseID)
	log.Printf("  Database Type: %s", cfg.DBAdapter)
	log.Printf("  Analyser Address: %s", cfg.AnalyserAddress)
	log.Printf("  Collection Interval: %v", cfg.CollectionInterval)

	// Create orchestrator to manage service lifecycle
	orch := orchestrator.NewOrchestrator(cfg)

	// Initialize all service connections
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
	// Exposes /health endpoint on port 8080
	health.StartHealthCheckServer("8080")

	// Start metric collection loop in background goroutine
	go func() {
		if err := orch.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	// Block until shutdown signal received
	<-sigChan
	log.Printf("Shutdown signal received, initiating graceful shutdown...")

	// Cancel context to stop collection loop
	cancel()

	// Close all connections and cleanup resources
	if err := orch.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("Collector stopped successfully")
}
