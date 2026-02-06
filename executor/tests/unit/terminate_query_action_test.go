package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestTerminateQueryAction_ExecuteSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsQueryTermination: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-1",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 12345, "app_user", true)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, int32(12345), result.Changes["pid"])
	assert.Equal(t, "app_user", result.Changes["username"])
	assert.False(t, result.CanRollback)
}

func TestTerminateQueryAction_ExecuteFailure(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:   database.Capabilities{SupportsQueryTermination: true},
		TerminateError: errors.New("permission denied"),
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-2",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 12345, "app_user", true)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "permission denied")
}

func TestTerminateQueryAction_GracefulFallbackToForceful(t *testing.T) {
	callCount := 0
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsQueryTermination: true},
		TerminateFunc: func(pid int32, graceful bool) error {
			callCount++
			if graceful {
				return errors.New("cancel failed")
			}
			return nil // Forceful succeeds
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-3",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 12345, "app_user", true)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, 2, callCount, "Should try graceful then forceful")
	assert.Contains(t, result.Changes["method"], "fallback")
}

func TestTerminateQueryAction_ValidateNoSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsQueryTermination: false},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-4",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 12345, "app_user", true)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrActionNotSupported, err)
}

func TestTerminateQueryAction_ValidateInvalidPID(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsQueryTermination: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-5",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 0, "app_user", true)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PID")
}

func TestTerminateQueryAction_RollbackIsNoop(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsQueryTermination: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-6",
		ActionType: "terminate_query",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewTerminateQueryAction(metadata, mock, 12345, "app_user", true)

	err := action.Rollback(context.Background())

	assert.NoError(t, err)
}
