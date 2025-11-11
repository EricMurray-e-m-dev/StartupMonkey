package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBConnectionString string
	DBAdapter          string
	DatabaseID         string
	DatabaseName       string
	AnalyserAddress    string
	CollectionInterval time.Duration
	NatsURL            string
}

func Load() (*Config, error) {

	envPath := filepath.Join("..", ".env")
	_ = godotenv.Load(envPath)

	config := &Config{
		DBConnectionString: os.Getenv("DB_CONNECTION_STRING"),
		DBAdapter:          os.Getenv("DB_ADAPTER"),
		DatabaseID:         os.Getenv("DATABASE_ID"),
		DatabaseName:       os.Getenv("DATABASE_NAME"),
		AnalyserAddress:    os.Getenv("ANALYSER_ADDRESS"),
		NatsURL:            os.Getenv("NATS_URL"),
	}

	intervalStr := os.Getenv("COLLECTION_INTERVAL")
	if intervalStr == "" {
		intervalStr = "30s"
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid COLLECTION_INTERVAL: %w", err)
	}
	config.CollectionInterval = interval

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.DBConnectionString == "" {
		return fmt.Errorf("DB_CONNECTION_STRING is required")
	}

	if c.DBAdapter == "" {
		return fmt.Errorf("DB_ADAPTER is required")
	}

	if c.AnalyserAddress == "" {
		return fmt.Errorf("ANALYSER_ADDRESS is required")
	}

	if c.DatabaseID == "" {
		return fmt.Errorf("DATABASE_ID is required")
	}
	return nil
}
