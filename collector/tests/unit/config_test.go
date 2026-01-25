package unit

import (
	"os"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_ValidateBootstrap_MissingFields(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		errMsg string
	}{
		{
			name: "missing analyser address",
			config: config.Config{
				KnowledgeAddress:   "localhost:50053",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "ANALYSER_ADDRESS",
		},
		{
			name: "missing knowledge address",
			config: config.Config{
				AnalyserAddress:    "localhost:50051",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "KNOWLEDGE_ADDRESS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateBootstrap()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_ValidateFull_MissingFields(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		errMsg string
	}{
		{
			name: "missing connection string",
			config: config.Config{
				DBAdapter:          "postgres",
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				DatabaseID:         "test",
				DatabaseName:       "testdb",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "DB_CONNECTION_STRING",
		},
		{
			name: "missing adapter",
			config: config.Config{
				DBConnectionString: "conn",
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				DatabaseID:         "test",
				DatabaseName:       "testdb",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "DB_ADAPTER",
		},
		{
			name: "missing database ID",
			config: config.Config{
				DBConnectionString: "conn",
				DBAdapter:          "postgres",
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				DatabaseName:       "testdb",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "DATABASE_ID",
		},
		{
			name: "missing database name",
			config: config.Config{
				DBConnectionString: "conn",
				DBAdapter:          "postgres",
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				DatabaseID:         "test",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "DATABASE_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateFull()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_ValidateBootstrap_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 30 * time.Second,
	}

	err := cfg.ValidateBootstrap()

	assert.NoError(t, err)
}

func TestConfig_ValidateFull_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		DBConnectionString: "postgres://test@localhost/test",
		DBAdapter:          "postgres",
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		DatabaseID:         "test-db",
		DatabaseName:       "test",
		CollectionInterval: 30 * time.Second,
	}

	err := cfg.ValidateFull()

	assert.NoError(t, err)
}

func TestConfig_LoadBootstrap_WithDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("DB_CONNECTION_STRING")
	os.Unsetenv("DB_ADAPTER")
	os.Unsetenv("DATABASE_ID")
	os.Unsetenv("DATABASE_NAME")

	// Set required bootstrap fields
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")

	defer os.Clearenv()

	cfg, err := config.LoadBootstrap()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost:50051", cfg.AnalyserAddress)
	assert.Equal(t, "localhost:50053", cfg.KnowledgeAddress)
	assert.Equal(t, 10*time.Second, cfg.CollectionInterval) // Default
	// DB fields should be empty (loaded from Knowledge later)
	assert.Empty(t, cfg.DBConnectionString)
	assert.Empty(t, cfg.DBAdapter)
	assert.Empty(t, cfg.DatabaseID)
}

func TestConfig_LoadBootstrap_CustomInterval(t *testing.T) {
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("COLLECTION_INTERVAL", "30s")

	defer os.Clearenv()

	cfg, err := config.LoadBootstrap()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.CollectionInterval)
}

func TestConfig_ValidateBootstrap_InvalidInterval(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 500 * time.Millisecond, // Too short
	}

	err := cfg.ValidateBootstrap()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "COLLECTION_INTERVAL")
}

func TestConfig_SetDatabaseConfig(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 10 * time.Second,
	}

	cfg.SetDatabaseConfig(
		"postgres://user:pass@localhost:5432/mydb",
		"postgres",
		"my-db-id",
		"My Database",
	)

	assert.Equal(t, "postgres://user:pass@localhost:5432/mydb", cfg.DBConnectionString)
	assert.Equal(t, "postgres", cfg.DBAdapter)
	assert.Equal(t, "my-db-id", cfg.DatabaseID)
	assert.Equal(t, "My Database", cfg.DatabaseName)

	// Should pass full validation now
	err := cfg.ValidateFull()
	assert.NoError(t, err)
}
