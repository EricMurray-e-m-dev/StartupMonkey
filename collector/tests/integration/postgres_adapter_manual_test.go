package integration

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/stretchr/testify/assert"
)

func TestPostgresAdapter_ManualConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Local Postgres instance for testing
	connString := "postgres://ericmurray@localhost:5432/postgres"

	// Create our adapter
	adapter := adapter.NewPostgresAdapter(connString)

	// Test Connect
	err := adapter.Connect()
	assert.NoError(t, err, "Connect should succeed")
	defer adapter.Close()

	// Test HealthCheck
	err = adapter.HealthCheck()
	assert.NoError(t, err, "Health check should succeed")

	// Test CollectMetrics
	metrics, err := adapter.CollectMetrics()
	assert.NoError(t, err, "CollectMetrics should succeed")
	assert.NotNil(t, metrics, "Metrics should not be nil")

	// Verify metrics are populated
	assert.Greater(t, metrics.MaxConnections, int32(0), "MaxConnections should be positive")
	assert.GreaterOrEqual(t, metrics.ActiveConnections, int32(0), "ActiveConnections should be non-negative")
	assert.NotEmpty(t, metrics.ExtendedMetrics, "ExtendedMetrics should be populated")

	// Manually log results
	t.Logf("Active Connections: %d", metrics.ActiveConnections)
	t.Logf("Idle Connections: %d", metrics.IdleConnections)
	t.Logf("Max Connections: %d", metrics.MaxConnections)
	t.Logf("Cache Hit Rate: %.2f%%", metrics.CacheHitRate)
	t.Logf("DB Size: %.2f MB", metrics.ExtendedMetrics["pg.database_size_mb"])

}
