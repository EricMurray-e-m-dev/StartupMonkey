package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the Knowledge service.
type Config struct {
	// Service addresses
	GRPCPort   string
	HealthPort string

	// Redis connection
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Feature flags
	EnableMetrics bool
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try multiple .env locations
	envPaths := []string{
		".env",
		"../.env",
		"../../.env",
		"/app/.env", // Docker
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
		GRPCPort:   getEnvOrDefault("GRPC_PORT", "50053"),
		HealthPort: getEnvOrDefault("HEALTH_PORT", "8083"),

		// Redis connection with defaults
		RedisAddr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       parseIntOrDefault("REDIS_DB", 0),

		// Feature flags
		EnableMetrics: getEnvOrDefault("ENABLE_METRICS", "false") == "true",
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

	if c.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
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
