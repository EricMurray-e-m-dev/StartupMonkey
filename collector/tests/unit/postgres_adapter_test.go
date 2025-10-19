package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/stretchr/testify/assert"
)

func TestNewPostgresAdapter(t *testing.T) {
	connString := "postgres://test@localhost/testdb"
	databaseID := "test-db-1"

	adapter := adapter.NewPostgresAdapter(connString, databaseID)

	assert.NotNil(t, adapter)
}

func TestPostgresAdapter_HealthCheck_NotConnected(t *testing.T) {
	pgAdapter := adapter.NewPostgresAdapter("postgres://test@localhost/testdb", "test-db-1")

	err := pgAdapter.HealthCheck()

	assert.Error(t, err)
	assert.Equal(t, adapter.ErrNotConnected, err)
}

func TestPostgresAdapter_Close_SafeMultipleCalls(t *testing.T) {
	pgAdapter := adapter.NewPostgresAdapter("postgres://test@localhost/testdb", "test-db-1")

	err := pgAdapter.Close()
	assert.NoError(t, err)

	err = pgAdapter.Close()
	assert.NoError(t, err)
}

func TestPostgresAdapter_CollectMetrics_NotConnected(t *testing.T) {
	pgAdapter := adapter.NewPostgresAdapter("postgres://test@localhost/testdb", "test-db-1")

	metrics, err := pgAdapter.CollectMetrics()

	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Equal(t, adapter.ErrNotConnected, err)
}
