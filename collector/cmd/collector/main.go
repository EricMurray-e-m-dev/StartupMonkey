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

func main() {
	log.Printf("StartupMonkey Collector starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded")
	log.Printf("  Knowledge Address: %s", cfg.KnowledgeAddress)
	log.Printf("  Analyser Address: %s", cfg.AnalyserAddress)
	log.Printf("  Collection Interval: %v", cfg.CollectionInterval)
	log.Printf("  Sync Interval: %v", cfg.SyncInterval)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(cfg)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for shutdown signals in goroutine (fixes race condition)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Printf("Shutdown signal received...")
		cancel()
	}()

	// Start health check server
	health.StartHealthCheckServer("8080")

	// Initialize orchestrator (will wait for databases from Knowledge)
	if err := orch.Start(ctx); err != nil {
		if err == context.Canceled {
			log.Printf("Startup cancelled by user")
		} else {
			log.Fatalf("Failed to start orchestrator: %v", err)
		}
		orch.Stop()
		health.StopHealthCheckServer()
		return
	}

	// Start metric collection (blocks until context cancelled)
	if err := orch.Run(ctx); err != nil && err != context.Canceled {
		log.Printf("Orchestrator error: %v", err)
	}

	health.StopHealthCheckServer()
	if err := orch.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("Collector stopped successfully")
}
