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

	// Load bootstrap configuration (service addresses only)
	cfg, err := config.LoadBootstrap()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Bootstrap configuration loaded")
	log.Printf("  Knowledge Address: %s", cfg.KnowledgeAddress)
	log.Printf("  Analyser Address: %s", cfg.AnalyserAddress)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(cfg)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start health check server
	health.StartHealthCheckServer("8080")

	// Initialize orchestrator (will wait for config from Knowledge)
	if err := orch.Start(ctx); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Start metric collection in background
	go func() {
		if err := orch.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	// Block until shutdown signal
	<-sigChan
	log.Printf("Shutdown signal received...")

	cancel()

	if err := orch.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("Collector stopped successfully")
}
