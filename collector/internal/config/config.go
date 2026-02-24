// Package config provides configuration loading for the Collector service.
package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds bootstrap configuration for the Collector service.
// Database connections are managed dynamically via Knowledge service.
type Config struct {
	// Service addresses
	AnalyserAddress  string
	NatsURL          string
	KnowledgeAddress string

	// Operational settings
	CollectionInterval time.Duration
	SyncInterval       time.Duration // How often to check for database changes

	// Feature flags
	EnableMetricsPublishing bool
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	envPaths := []string{
		".env",
		"../.env",
		"/app/.env",
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded config from: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("No .env file found, using environment variables")
	}

	config := &Config{
		AnalyserAddress:         getEnvOrDefault("ANALYSER_ADDRESS", "localhost:50051"),
		NatsURL:                 getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		KnowledgeAddress:        getEnvOrDefault("KNOWLEDGE_ADDRESS", "localhost:50053"),
		EnableMetricsPublishing: getEnvOrDefault("ENABLE_METRICS_PUBLISHING", "true") == "true",
	}

	// Parse collection interval
	intervalStr := getEnvOrDefault("COLLECTION_INTERVAL", "10s")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid COLLECTION_INTERVAL: %w", err)
	}
	config.CollectionInterval = interval

	// Parse sync interval (how often to check for new/removed databases)
	syncStr := getEnvOrDefault("SYNC_INTERVAL", "30s")
	syncInterval, err := time.ParseDuration(syncStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SYNC_INTERVAL: %w", err)
	}
	config.SyncInterval = syncInterval

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.AnalyserAddress == "" {
		return fmt.Errorf("ANALYSER_ADDRESS is required")
	}

	if c.KnowledgeAddress == "" {
		return fmt.Errorf("KNOWLEDGE_ADDRESS is required")
	}

	if c.CollectionInterval < 1*time.Second {
		return fmt.Errorf("COLLECTION_INTERVAL must be at least 1 second")
	}

	if c.SyncInterval < 5*time.Second {
		return fmt.Errorf("SYNC_INTERVAL must be at least 5 seconds")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
