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

// MockDatabaseAdapter implements database.DatabaseAdapter for testing
type MockDatabaseAdapter struct {
	VacuumCalled    bool
	VacuumTableName string
	VacuumError     error
	DeadTuples      int64
	DeadTuplesError error
	Capabilities    database.Capabilities
}

func (m *MockDatabaseAdapter) CreateIndex(ctx context.Context, params database.IndexParams) error {
	return nil
}

func (m *MockDatabaseAdapter) DropIndex(ctx context.Context, indexName string) error {
	return nil
}

func (m *MockDatabaseAdapter) IndexExists(ctx context.Context, indexName string) (bool, error) {
	return false, nil
}

func (m *MockDatabaseAdapter) GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error) {
	return nil, nil
}

func (m *MockDatabaseAdapter) SetConfig(ctx context.Context, changes map[string]string) error {
	return nil
}

func (m *MockDatabaseAdapter) GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]database.SlowQuery, error) {
	return nil, nil
}

func (m *MockDatabaseAdapter) VacuumTable(ctx context.Context, tableName string) error {
	m.VacuumCalled = true
	m.VacuumTableName = tableName
	return m.VacuumError
}

func (m *MockDatabaseAdapter) GetDeadTuples(ctx context.Context, tableName string) (int64, error) {
	if m.DeadTuplesError != nil {
		return 0, m.DeadTuplesError
	}
	return m.DeadTuples, nil
}

func (m *MockDatabaseAdapter) GetCapabilities() database.Capabilities {
	return m.Capabilities
}

func (m *MockDatabaseAdapter) Close() error {
	return nil
}

func TestVacuumTableAction_ExecuteSuccess(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: true},
		DeadTuples:   5000,
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-1",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "posts")

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.True(t, mock.VacuumCalled)
	assert.Equal(t, "posts", mock.VacuumTableName)
	assert.False(t, result.CanRollback)
}

func TestVacuumTableAction_ExecuteFailure(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: true},
		VacuumError:  errors.New("vacuum failed: table locked"),
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-2",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "posts")

	result, err := action.Execute(context.Background())

	assert.NoError(t, err) // Action returns result, not error
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "vacuum failed")
}

func TestVacuumTableAction_ValidateNoVacuumSupport(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: false},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-3",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "posts")

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Equal(t, database.ErrActionNotSupported, err)
}

func TestVacuumTableAction_ValidateMissingTableName(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-4",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "")

	err := action.Validate(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "table name is required")
}

func TestVacuumTableAction_RollbackIsNoop(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: true},
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-5",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "posts")

	err := action.Rollback(context.Background())

	assert.NoError(t, err)
}

func TestVacuumTableAction_RecordsTupleStats(t *testing.T) {
	mock := &MockDatabaseAdapter{
		Capabilities: database.Capabilities{SupportsVacuum: true},
		DeadTuples:   10000,
	}

	metadata := &models.ActionMetadata{
		ActionID:   "test-action-6",
		ActionType: "vacuum_table",
		DatabaseID: "test-db",
		CreatedAt:  time.Now(),
	}

	action := actions.NewVacuumTableAction(metadata, mock, "posts")

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, "posts", result.Changes["table_name"])
	assert.Equal(t, "VACUUM ANALYZE", result.Changes["operation"])
}
