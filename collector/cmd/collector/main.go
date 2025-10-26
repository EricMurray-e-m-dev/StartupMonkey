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

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded")
	log.Printf("	Database: %s", cfg.DBAdapter)
	log.Printf("	Analyser: %s", cfg.AnalyserAddress)
	log.Printf("	Collection Tick: %v", cfg.CollectionInterval)

	orch := orchestrator.NewOrchestrator(cfg)

	if err := orch.Start(); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	health.StartHealthCheckServer("8080")

	go func() {
		if err := orch.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	<-sigChan
	log.Printf("Shutdown signal received")

	cancel()

	if err := orch.Stop(); err != nil {
		log.Printf("Error during stop: %v", err)
	}

	log.Printf("Collector stopped")
}
