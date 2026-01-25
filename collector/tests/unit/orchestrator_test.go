package unit

import (
	"context"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func TestNewOrchestrator(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 30 * time.Second,
	}

	orch := orchestrator.NewOrchestrator(cfg)

	assert.NotNil(t, orch)
}

func TestOrchestrator_Start_FailsWithoutKnowledge(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:99999", // Invalid port
		CollectionInterval: 30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	orch := orchestrator.NewOrchestrator(cfg)
	err := orch.Start(ctx)

	// Should fail to connect to Knowledge or timeout waiting for config
	assert.Error(t, err)
}

func TestOrchestrator_Stop_SafeWhenNotStarted(t *testing.T) {
	cfg := &config.Config{
		AnalyserAddress:    "localhost:50051",
		KnowledgeAddress:   "localhost:50053",
		CollectionInterval: 30 * time.Second,
	}

	orch := orchestrator.NewOrchestrator(cfg)
	err := orch.Stop()

	assert.NoError(t, err)
}
