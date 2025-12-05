package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/orchestrator"
)

// main is the entry point for the Knowledge service.
//
// The Knowledge service is the central state store for StartupMonkey:
//   - Stores database connection strings (for Executor autonomous actions)
//   - Tracks detection state (for Analyser deduplication logic)
//   - Manages action status (for Dashboard real-time visibility)
//   - Provides system-wide statistics (databases, detections, actions)
//
// All state is persisted in Redis for reliability and performance.
//
// Lifecycle:
//  1. Load configuration from environment variables
//  2. Initialize orchestrator with Redis client and servers
//  3. Start gRPC server (Knowledge API) and health check server
//  4. Listen for shutdown signals (SIGINT, SIGTERM)
//  5. Gracefully close all connections on shutdown
func main() {
	log.Printf("StartupMonkey Knowledge (Brain) starting...")

	// Load configuration from environment variables and .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  gRPC Port: %s", cfg.GRPCPort)
	log.Printf("  Health Port: %s", cfg.HealthPort)
	log.Printf("  Redis Address: %s", cfg.RedisAddr)
	log.Printf("  Redis DB: %d", cfg.RedisDB)
	log.Printf("  Metrics Enabled: %v", cfg.EnableMetrics)

	// Create orchestrator to manage service lifecycle
	orch := orchestrator.NewOrchestrator(cfg)

	// Initialize Redis connection and servers
	if err := orch.Start(); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Setup graceful shutdown handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for shutdown signals (Ctrl+C, Docker stop, k8s termination)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC and health check servers in background goroutine
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

	log.Printf("Knowledge service stopped successfully")
}
