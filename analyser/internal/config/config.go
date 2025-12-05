package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the Analyser service.
type Config struct {
	// Service addresses
	GRPCPort         string
	HealthPort       string
	NatsURL          string
	KnowledgeAddress string

	// Detection thresholds (configurable per detector)
	Thresholds DetectionThresholds

	// Feature flags
	EnableAllDetectors bool
}

// DetectionThresholds contains configurable thresholds for each detector.
// These can be adjusted based on environment (dev/staging/prod) or via Dashboard in future.
type DetectionThresholds struct {
	// Connection Pool Detector
	ConnectionPoolWarning  float64 // e.g., 0.7 = 70% utilization
	ConnectionPoolCritical float64 // e.g., 0.9 = 90% utilization

	// Missing Index Detector
	SequentialScanThreshold      int32   // Minimum seq scans to trigger
	SequentialScanDeltaThreshold float64 // Delta increase to trigger

	// High Latency Detector
	P95LatencyThresholdMs float64 // P95 latency in milliseconds
	P99LatencyThresholdMs float64 // P99 latency in milliseconds

	// Cache Miss Detector
	CacheHitRateThreshold float64 // Minimum cache hit rate (0.0-1.0)
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try multiple .env locations
	envPaths := []string{
		".env",
		"../.env",
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
		GRPCPort:         getEnvOrDefault("GRPC_PORT", "50051"),
		HealthPort:       getEnvOrDefault("HEALTH_PORT", "8081"),
		NatsURL:          getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		KnowledgeAddress: getEnvOrDefault("KNOWLEDGE_ADDRESS", "localhost:50053"),

		// Feature flags
		EnableAllDetectors: getEnvOrDefault("ENABLE_ALL_DETECTORS", "true") == "true",

		// Default thresholds (can be overridden by env vars)
		Thresholds: DetectionThresholds{
			// Connection Pool (changed from 0.8 to 0.1 for local testing)
			ConnectionPoolWarning:  parseFloatOrDefault("THRESHOLD_CONNECTION_POOL_WARNING", 0.7),
			ConnectionPoolCritical: parseFloatOrDefault("THRESHOLD_CONNECTION_POOL_CRITICAL", 0.1),

			// Missing Index
			SequentialScanThreshold:      int32(parseIntOrDefault("THRESHOLD_SEQ_SCAN", 1)),
			SequentialScanDeltaThreshold: parseFloatOrDefault("THRESHOLD_SEQ_SCAN_DELTA", 10.0),

			// High Latency
			P95LatencyThresholdMs: parseFloatOrDefault("THRESHOLD_P95_LATENCY_MS", 500.0),
			P99LatencyThresholdMs: parseFloatOrDefault("THRESHOLD_P99_LATENCY_MS", 1000.0),

			// Cache Miss
			CacheHitRateThreshold: parseFloatOrDefault("THRESHOLD_CACHE_HIT_RATE", 0.8),
		},
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

	if c.KnowledgeAddress == "" {
		return fmt.Errorf("KNOWLEDGE_ADDRESS is required")
	}

	// Validate threshold ranges
	if c.Thresholds.ConnectionPoolWarning < 0 || c.Thresholds.ConnectionPoolWarning > 1 {
		return fmt.Errorf("CONNECTION_POOL_WARNING must be between 0 and 1")
	}

	if c.Thresholds.CacheHitRateThreshold < 0 || c.Thresholds.CacheHitRateThreshold > 1 {
		return fmt.Errorf("CACHE_HIT_RATE_THRESHOLD must be between 0 and 1")
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

func parseFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var result float64
		if _, err := fmt.Sscanf(value, "%f", &result); err == nil {
			return result
		}
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
