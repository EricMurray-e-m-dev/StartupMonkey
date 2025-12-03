package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/orchestrator"
)

// main is the entry point for the Analyser service.
//
// The Analyser is responsible for:
//   - Receiving normalized metrics from the Collector service via gRPC
//   - Running detection algorithms to identify performance issues
//   - Registering detections with Knowledge for deduplication
//   - Publishing detections to NATS for Executor to consume
//   - Subscribing to action completion events to mark detections as resolved
//
// Lifecycle:
//  1. Load configuration from environment variables
//  2. Initialize orchestrator with detection engine and service connections
//  3. Start health check server (port 8081)
//  4. Start gRPC server to receive metrics
//  5. Listen for shutdown signals (SIGINT, SIGTERM)
//  6. Gracefully close all connections on shutdown
func main() {
	log.Printf("StartupMonkey Analyser starting...")

	// Load configuration from environment variables and .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  gRPC Port: %s", cfg.GRPCPort)
	log.Printf("  Health Port: %s", cfg.HealthPort)
	log.Printf("  NATS URL: %s", cfg.NatsURL)
	log.Printf("  Knowledge Address: %s", cfg.KnowledgeAddress)
	log.Printf("  Detectors Enabled: %v", cfg.EnableAllDetectors)

	// Log configured thresholds
	log.Printf("Detection Thresholds:")
	log.Printf("  Connection Pool Critical: %.0f%%", cfg.Thresholds.ConnectionPoolCritical*100)
	log.Printf("  Sequential Scan: %d (delta: %.1f)",
		cfg.Thresholds.SequentialScanThreshold,
		cfg.Thresholds.SequentialScanDeltaThreshold)
	log.Printf("  P95 Latency: %.0fms", cfg.Thresholds.P95LatencyThresholdMs)
	log.Printf("  Cache Hit Rate: %.0f%%", cfg.Thresholds.CacheHitRateThreshold*100)

	// Create orchestrator to manage service lifecycle
	orch := orchestrator.NewOrchestrator(cfg)

	// Initialize all service connections and detection engine
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
	// Exposes /health endpoint on configured port (default: 8081)
	health.StartHealthCheckServer(cfg.HealthPort)

	// Start gRPC server in background goroutine
	go func() {
		if err := orch.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	// Block until shutdown signal received
	<-sigChan
	log.Printf("Shutdown signal received, initiating graceful shutdown...")

	// Cancel context to stop gRPC server
	cancel()

	// Close all connections and cleanup resources
	if err := orch.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("Analyser stopped successfully")
}
