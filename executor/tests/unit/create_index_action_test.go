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

func TestCreateIndexAction_ExecuteSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:           true,
			SupportsConcurrentIndexes: true,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-1",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, "posts", result.Changes["table_name"])
	assert.Equal(t, []string{"user_id"}, result.Changes["column_names"])
	assert.True(t, result.CanRollback)
	assert.True(t, mock.CreateIndexCalled)
}

func TestCreateIndexAction_ExecuteSuccessUniqueIndex(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:     true,
			SupportsUniqueIndex: true,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-2",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "users", []string{"email"}, true)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, true, result.Changes["unique"])
}

func TestCreateIndexAction_ExecuteFailure(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:     database.Capabilities{SupportsIndexes: true},
		CreateIndexError: errors.New("failed to create index: permission denied"),
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-3",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "permission denied")
	assert.False(t, result.CanRollback)
}

func TestCreateIndexAction_ValidateNoIndexSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsIndexes: false},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-4",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrActionNotSupported, err)
}

func TestCreateIndexAction_ValidateMultiColumnNoSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:          true,
			SupportsMultiColumnIndex: false,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-5",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id", "created_at"}, false)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrActionNotSupported, err)
}

func TestCreateIndexAction_ValidateUniqueNoSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:     true,
			SupportsUniqueIndex: false,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-6",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "users", []string{"email"}, true)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrActionNotSupported, err)
}

func TestCreateIndexAction_ValidateIndexAlreadyExists(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:     database.Capabilities{SupportsIndexes: true},
		IndexExistsValue: true,
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-7",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrIndexAlreadyExists, err)
}

func TestCreateIndexAction_ValidateIndexExistsError(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:     database.Capabilities{SupportsIndexes: true},
		IndexExistsError: errors.New("connection failed"),
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-8",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestCreateIndexAction_RollbackSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsIndexes: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-9",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	// Execute first to set indexCreated = true
	_, err := action.Execute(context.Background())
	assert.NoError(t, err)

	// Now index exists for rollback check
	mock.IndexExistsValue = true

	err = action.Rollback(context.Background())

	assert.NoError(t, err)
	assert.True(t, mock.DropIndexCalled)
}

func TestCreateIndexAction_RollbackNotCreated(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsIndexes: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-10",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	// Don't execute, so indexCreated = false
	err := action.Rollback(context.Background())

	assert.NoError(t, err)
	assert.False(t, mock.DropIndexCalled)
}

func TestCreateIndexAction_RollbackIndexNotFound(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:     database.Capabilities{SupportsIndexes: true},
		IndexExistsValue: false,
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-11",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	// Execute first
	_, _ = action.Execute(context.Background())

	// Index doesn't exist anymore (maybe dropped manually)
	mock.IndexExistsValue = false

	err := action.Rollback(context.Background())

	assert.NoError(t, err)
	assert.False(t, mock.DropIndexCalled)
}

func TestCreateIndexAction_RollbackDropError(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities:     database.Capabilities{SupportsIndexes: true},
		IndexExistsValue: false, // Start false so Execute succeeds
		DropIndexError:   errors.New("cannot drop index: in use"),
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-12",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	// Execute first to set indexCreated = true
	result, _ := action.Execute(context.Background())
	assert.Equal(t, models.StatusCompleted, result.Status)

	// Now index exists for rollback check
	mock.IndexExistsValue = true

	err := action.Rollback(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot drop index")
}

func TestCreateIndexAction_GetMetadata(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsIndexes: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-13",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	result := action.GetMetadata()

	assert.Equal(t, metadata, result)
	assert.Equal(t, "test-action-13", result.ActionID)
}

func TestCreateIndexAction_ConcurrentFlagSet(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:           true,
			SupportsConcurrentIndexes: true,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-14",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, true, result.Changes["concurrent"])
}

func TestCreateIndexAction_ConcurrentFlagNotSet(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{
			SupportsIndexes:           true,
			SupportsConcurrentIndexes: false,
		},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-15",
		ActionType: "create_index",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewCreateIndexAction(metadata, mock, "posts", []string{"user_id"}, false)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, false, result.Changes["concurrent"])
}
