package unit

import (
	"os"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate_MissingFields(t *testing.T) {
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
			name: "missing analyser address",
			config: config.Config{
				DBConnectionString: "conn",
				DBAdapter:          "postgres",
				KnowledgeAddress:   "localhost:50053",
				DatabaseID:         "test",
				DatabaseName:       "testdb",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "ANALYSER_ADDRESS",
		},
		{
			name: "missing knowledge address",
			config: config.Config{
				DBConnectionString: "conn",
				DBAdapter:          "postgres",
				AnalyserAddress:    "localhost:50051",
				DatabaseID:         "test",
				DatabaseName:       "testdb",
				CollectionInterval: 30 * time.Second,
			},
			errMsg: "KNOWLEDGE_ADDRESS",
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
			err := tt.config.Validate()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		DBConnectionString: "postgres://test@localhost/test",
		DBAdapter:          "postgres",
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		DatabaseID:         "test-db",
		DatabaseName:       "test",
		CollectionInterval: 30 * time.Second,
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

func TestConfig_Load_WithDefaults(t *testing.T) {
	// Set all required fields
	os.Setenv("DB_CONNECTION_STRING", "test-conn")
	os.Setenv("DB_ADAPTER", "postgres")
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("DATABASE_ID", "test-db")
	os.Setenv("DATABASE_NAME", "testdb")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-conn", cfg.DBConnectionString)
	assert.Equal(t, "postgres", cfg.DBAdapter)
	assert.Equal(t, "localhost:50051", cfg.AnalyserAddress)
	assert.Equal(t, "localhost:50053", cfg.KnowledgeAddress)
	assert.Equal(t, "test-db", cfg.DatabaseID)
	assert.Equal(t, "testdb", cfg.DatabaseName)
	assert.Equal(t, 10*time.Second, cfg.CollectionInterval) // Default from Load()
}

func TestConfig_Load_CustomInterval(t *testing.T) {
	// Set all required fields
	os.Setenv("DB_CONNECTION_STRING", "test-conn")
	os.Setenv("DB_ADAPTER", "postgres")
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("DATABASE_ID", "test-db")
	os.Setenv("DATABASE_NAME", "testdb")
	os.Setenv("COLLECTION_INTERVAL", "30s")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.CollectionInterval)
}

func TestConfig_Validate_InvalidInterval(t *testing.T) {
	cfg := &config.Config{
		DBConnectionString: "postgres://test@localhost/test",
		DBAdapter:          "postgres",
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		DatabaseID:         "test-db",
		DatabaseName:       "test",
		CollectionInterval: 500 * time.Millisecond, // Too short
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "COLLECTION_INTERVAL")
}
