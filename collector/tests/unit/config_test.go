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
			name:   "missing connection string",
			config: config.Config{DBAdapter: "postgres", AnalyserAddress: "localhost:50051", DatabaseID: "test"},
			errMsg: "DB_CONNECTION_STRING",
		},
		{
			name:   "missing adapter",
			config: config.Config{DBConnectionString: "conn", AnalyserAddress: "localhost:50051", DatabaseID: "test"},
			errMsg: "DB_ADAPTER",
		},
		{
			name:   "missing analyser address",
			config: config.Config{DBConnectionString: "conn", DBAdapter: "postgres", DatabaseID: "test"},
			errMsg: "ANALYSER_ADDRESS",
		},
		{
			name:   "missing database ID",
			config: config.Config{DBConnectionString: "conn", DBAdapter: "postgres", AnalyserAddress: "localhost:50051"},
			errMsg: "DATABASE_ID",
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
		DatabaseID:         "test-db",
		DatabaseName:       "test",
		CollectionInterval: 30 * time.Second,
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

func TestConfig_Load_WithDefaults(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "test-conn")
	os.Setenv("DB_ADAPTER", "postgres")
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("DATABASE_ID", "test-db")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.CollectionInterval)
}
