package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewTuneConfigAction_Success(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning: true,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)

	assert.NoError(t, err)
	assert.NotNil(t, action)
}

func TestNewTuneConfigAction_NoConfigTuningSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning: false,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)

	assert.Error(t, err)
	assert.Nil(t, action)
	assert.Contains(t, err.Error(), "does not support config tuning")
}

func TestTuneConfigAction_ExecuteWithChanges(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem":             "4MB",
			"effective_cache_size": "4GB",
			"random_page_cost":     "4",
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.True(t, result.CanRollback)
	assert.True(t, mock.SetConfigCalled)

	configChanges := result.Changes["config_changes"].(map[string]string)
	assert.Equal(t, "16MB", configChanges["work_mem"])
	assert.Equal(t, "8GB", configChanges["effective_cache_size"])
	assert.Equal(t, "1.1", configChanges["random_page_cost"])
}

func TestTuneConfigAction_ExecuteNoChangesNeeded(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem":             "16MB",
			"effective_cache_size": "16GB",
			"random_page_cost":     "1.1",
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.False(t, result.CanRollback)
	assert.False(t, mock.SetConfigCalled)
	assert.Contains(t, result.Message, "already optimal")
}

func TestTuneConfigAction_ExecuteGetConfigError(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigError: errors.New("connection lost"),
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	result, err := action.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestTuneConfigAction_ExecuteSetConfigError(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem":             "4MB",
			"effective_cache_size": "4GB",
			"random_page_cost":     "4",
		},
		SetConfigError: errors.New("permission denied"),
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	result, err := action.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestTuneConfigAction_ExecuteSlowQueriesErrorContinues(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem":             "4MB",
			"effective_cache_size": "4GB",
			"random_page_cost":     "4",
		},
		GetSlowQueriesError: errors.New("pg_stat_statements not available"),
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	result, err := action.Execute(context.Background())

	// Should still succeed even if slow queries fail
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
}

func TestTuneConfigAction_RollbackSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem":             "4MB",
			"effective_cache_size": "4GB",
			"random_page_cost":     "4",
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	// Execute first to set originalConfig
	_, err = action.Execute(context.Background())
	assert.NoError(t, err)

	// Reset mock tracking
	mock.SetConfigCalled = false

	err = action.Rollback(context.Background())

	assert.NoError(t, err)
	assert.True(t, mock.SetConfigCalled)
}

func TestTuneConfigAction_RollbackNoOriginalConfig(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	// Don't execute, so no originalConfig set
	err = action.Rollback(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no original config")
}

func TestTuneConfigAction_RollbackSetConfigError(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
		GetCurrentConfigResult: map[string]string{
			"work_mem": "4MB",
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	// Execute first
	_, err = action.Execute(context.Background())
	assert.NoError(t, err)

	// Now make SetConfig fail for rollback
	mock.SetConfigError = errors.New("connection lost")

	err = action.Rollback(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestTuneConfigAction_ValidateSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	err = action.Validate(context.Background())

	assert.NoError(t, err)
}

func TestTuneConfigAction_ValidateNoConfigTuning(t *testing.T) {
	// Create with tuning enabled, then change capabilities
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: true,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	// Simulate capability change
	mock.Capabilities.SupportsConfigTuning = false

	err = action.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support config tuning")
}

func TestTuneConfigAction_ValidateNoRuntimeChanges(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning:         true,
			SupportsRuntimeConfigChanges: false,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	err = action.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart required")
}

func TestTuneConfigAction_GetMetadata(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsConfigTuning: true,
		},
	}

	action, err := actions.NewTuneConfigAction("action-1", "detection-1", "test-db", "postgres", mock)
	assert.NoError(t, err)

	metadata := action.GetMetadata()

	assert.Equal(t, "action-1", metadata.ActionID)
	assert.Equal(t, "tune_config_high_latency", metadata.ActionType)
	assert.Equal(t, "test-db", metadata.DatabaseID)
	assert.Equal(t, "postgres", metadata.DatabaseType)
}
