package unit

import (
	"context"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
)

// MockDatabaseAdapter implements database.DatabaseAdapter for testing
type MockDatabaseAdapter struct {
	// Vacuum
	VacuumCalled    bool
	VacuumTableName string
	VacuumError     error
	DeadTuples      int64
	DeadTuplesError error

	// Terminate
	TerminateError error
	TerminateFunc  func(pid int32, graceful bool) error

	// Index
	CreateIndexCalled bool
	CreateIndexError  error
	DropIndexCalled   bool
	DropIndexError    error
	IndexExistsValue  bool
	IndexExistsError  error

	// Config
	GetCurrentConfigResult map[string]string
	GetCurrentConfigError  error
	SetConfigCalled        bool
	SetConfigError         error

	// Slow queries
	GetSlowQueriesResult []database.SlowQuery
	GetSlowQueriesError  error

	// Capabilities
	Capabilities database.Capabilities
}

func (m *MockDatabaseAdapter) CreateIndex(ctx context.Context, params database.IndexParams) error {
	m.CreateIndexCalled = true
	return m.CreateIndexError
}

func (m *MockDatabaseAdapter) DropIndex(ctx context.Context, indexName string) error {
	m.DropIndexCalled = true
	return m.DropIndexError
}

func (m *MockDatabaseAdapter) IndexExists(ctx context.Context, indexName string) (bool, error) {
	if m.IndexExistsError != nil {
		return false, m.IndexExistsError
	}
	return m.IndexExistsValue, nil
}

func (m *MockDatabaseAdapter) GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error) {
	if m.GetCurrentConfigError != nil {
		return nil, m.GetCurrentConfigError
	}
	return m.GetCurrentConfigResult, nil
}

func (m *MockDatabaseAdapter) SetConfig(ctx context.Context, changes map[string]string) error {
	m.SetConfigCalled = true
	return m.SetConfigError
}

func (m *MockDatabaseAdapter) GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]database.SlowQuery, error) {
	if m.GetSlowQueriesError != nil {
		return nil, m.GetSlowQueriesError
	}
	return m.GetSlowQueriesResult, nil
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

func (m *MockDatabaseAdapter) TerminateQuery(ctx context.Context, pid int32, graceful bool) error {
	if m.TerminateFunc != nil {
		return m.TerminateFunc(pid, graceful)
	}
	return m.TerminateError
}

func (m *MockDatabaseAdapter) GetCapabilities() database.Capabilities {
	return m.Capabilities
}

func (m *MockDatabaseAdapter) Close() error {
	return nil
}
