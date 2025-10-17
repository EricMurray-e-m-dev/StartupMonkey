package unit

import (
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func TestNewOrchestrator(t *testing.T) {
	cfg := &config.Config{
		DBConnectionString: "postgres://test@localhost/test",
		DBAdapter:          "postgres",
		AnalyserAddress:    "localhost:50051",
		DatabaseID:         "test-db",
		DatabaseName:       "test",
		CollectionInterval: 30 * time.Second,
	}

	orch := orchestrator.NewOrchestrator(cfg)

	assert.NotNil(t, orch)
}
