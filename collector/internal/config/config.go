package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database connection (loaded from Knowledge)
	DBConnectionString string
	DBAdapter          string
	DatabaseID         string
	DatabaseName       string

	// Service addresses (from env vars)
	AnalyserAddress  string
	NatsURL          string
	KnowledgeAddress string

	// Operational settings
	CollectionInterval time.Duration

	// Feature flags
	EnableMetricsPublishing bool
}

// LoadBootstrap loads only service addresses needed to connect to Knowledge.
// Database config will be fetched from Knowledge service.
func LoadBootstrap() (*Config, error) {
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
		// Service addresses (with defaults)
		AnalyserAddress:  getEnvOrDefault("ANALYSER_ADDRESS", "localhost:50051"),
		NatsURL:          getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		KnowledgeAddress: getEnvOrDefault("KNOWLEDGE_ADDRESS", "localhost:50053"),

		// Features
		EnableMetricsPublishing: getEnvOrDefault("ENABLE_METRICS_PUBLISHING", "true") == "true",
	}

	// Parse collection interval with default
	intervalStr := getEnvOrDefault("COLLECTION_INTERVAL", "10s")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid COLLECTION_INTERVAL: %w", err)
	}
	config.CollectionInterval = interval

	if err := config.ValidateBootstrap(); err != nil {
		return nil, err
	}

	return config, nil
}

// SetDatabaseConfig sets the database configuration fetched from Knowledge.
func (c *Config) SetDatabaseConfig(connString, dbType, dbID, dbName string) {
	c.DBConnectionString = connString
	c.DBAdapter = dbType
	c.DatabaseID = dbID
	c.DatabaseName = dbName
}

// ValidateBootstrap validates only the bootstrap configuration.
func (c *Config) ValidateBootstrap() error {
	required := map[string]string{
		"ANALYSER_ADDRESS":  c.AnalyserAddress,
		"KNOWLEDGE_ADDRESS": c.KnowledgeAddress,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}

	if c.CollectionInterval < 1*time.Second {
		return fmt.Errorf("COLLECTION_INTERVAL must be at least 1 second")
	}

	return nil
}

// ValidateFull validates the complete configuration including database.
func (c *Config) ValidateFull() error {
	if err := c.ValidateBootstrap(); err != nil {
		return err
	}

	required := map[string]string{
		"DB_CONNECTION_STRING": c.DBConnectionString,
		"DB_ADAPTER":           c.DBAdapter,
		"DATABASE_ID":          c.DatabaseID,
		"DATABASE_NAME":        c.DatabaseName,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}

	return nil
}

// Helper function for defaults
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
