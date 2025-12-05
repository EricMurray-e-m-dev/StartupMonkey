package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the Executor service.
type Config struct {
	// Service addresses
	GRPCPort         string
	HTTPPort         string
	HealthPort       string
	NatsURL          string
	KnowledgeAddress string

	// Action execution settings
	MaxConcurrentActions int
	ActionTimeout        int // seconds

	// Feature flags
	EnableAutoExecution bool
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try multiple .env locations
	envPaths := []string{
		".env",
		"../.env",
		"../../.env", // Executor is nested deeper
		"/app/.env",  // Docker
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
		// Service addresses with defaults
		GRPCPort:         getEnvOrDefault("GRPC_PORT", "50052"),
		HTTPPort:         getEnvOrDefault("HTTP_PORT", "8084"),
		HealthPort:       getEnvOrDefault("HEALTH_PORT", "8082"),
		NatsURL:          getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		KnowledgeAddress: getEnvOrDefault("KNOWLEDGE_ADDRESS", "localhost:50053"),

		// Action execution settings
		MaxConcurrentActions: parseIntOrDefault("MAX_CONCURRENT_ACTIONS", 10),
		ActionTimeout:        parseIntOrDefault("ACTION_TIMEOUT_SECONDS", 300), // 5 minutes

		// Feature flags
		EnableAutoExecution: getEnvOrDefault("ENABLE_AUTO_EXECUTION", "true") == "true",
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.GRPCPort == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}

	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT is required")
	}

	if c.KnowledgeAddress == "" {
		return fmt.Errorf("KNOWLEDGE_ADDRESS is required")
	}

	if c.MaxConcurrentActions < 1 {
		return fmt.Errorf("MAX_CONCURRENT_ACTIONS must be at least 1")
	}

	if c.ActionTimeout < 1 {
		return fmt.Errorf("ACTION_TIMEOUT_SECONDS must be at least 1")
	}

	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
