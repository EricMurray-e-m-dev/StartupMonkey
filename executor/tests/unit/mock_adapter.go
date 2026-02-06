package unit

import (
	"context"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
)

// MockDatabaseAdapter implements database.DatabaseAdapter for testing
type MockDatabaseAdapter struct {
	VacuumCalled    bool
	VacuumTableName string
	VacuumError     error
	DeadTuples      int64
	DeadTuplesError error
	Capabilities    database.Capabilities
	TerminateError  error
	TerminateFunc   func(pid int32, graceful bool) error
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
