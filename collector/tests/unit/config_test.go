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
			name: "missing analyser address",
			config: config.Config{
				KnowledgeAddress:   "localhost:50053",
				CollectionInterval: 30 * time.Second,
				SyncInterval:       30 * time.Second,
			},
			errMsg: "ANALYSER_ADDRESS",
		},
		{
			name: "missing knowledge address",
			config: config.Config{
				AnalyserAddress:    "localhost:50051",
				CollectionInterval: 30 * time.Second,
				SyncInterval:       30 * time.Second,
			},
			errMsg: "KNOWLEDGE_ADDRESS",
		},
		{
			name: "collection interval too short",
			config: config.Config{
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				CollectionInterval: 500 * time.Millisecond,
				SyncInterval:       30 * time.Second,
			},
			errMsg: "COLLECTION_INTERVAL",
		},
		{
			name: "sync interval too short",
			config: config.Config{
				AnalyserAddress:    "localhost:50051",
				KnowledgeAddress:   "localhost:50053",
				CollectionInterval: 10 * time.Second,
				SyncInterval:       2 * time.Second,
			},
			errMsg: "SYNC_INTERVAL",
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
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 30 * time.Second,
		SyncInterval:       30 * time.Second,
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

func TestConfig_Load_WithDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Clearenv()

	// Set required fields (or rely on defaults)
	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost:50051", cfg.AnalyserAddress)
	assert.Equal(t, "localhost:50053", cfg.KnowledgeAddress)
	assert.Equal(t, 10*time.Second, cfg.CollectionInterval) // Default
	assert.Equal(t, 30*time.Second, cfg.SyncInterval)       // Default
}

func TestConfig_Load_CustomIntervals(t *testing.T) {
	os.Clearenv()

	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("COLLECTION_INTERVAL", "15s")
	os.Setenv("SYNC_INTERVAL", "60s")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 15*time.Second, cfg.CollectionInterval)
	assert.Equal(t, 60*time.Second, cfg.SyncInterval)
}

func TestConfig_Load_InvalidCollectionInterval(t *testing.T) {
	os.Clearenv()

	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("COLLECTION_INTERVAL", "invalid")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "COLLECTION_INTERVAL")
}

func TestConfig_Load_InvalidSyncInterval(t *testing.T) {
	os.Clearenv()

	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("SYNC_INTERVAL", "invalid")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "SYNC_INTERVAL")
}

func TestConfig_Load_MetricsPublishingFlag(t *testing.T) {
	os.Clearenv()

	os.Setenv("ANALYSER_ADDRESS", "localhost:50051")
	os.Setenv("KNOWLEDGE_ADDRESS", "localhost:50053")
	os.Setenv("ENABLE_METRICS_PUBLISHING", "false")

	defer os.Clearenv()

	cfg, err := config.Load()

	assert.NoError(t, err)
	assert.False(t, cfg.EnableMetricsPublishing)
}
