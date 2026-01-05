package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestConnectionPoolDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(85)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire at 85pct usage.")
	assert.Equal(t, "connection_pool_exhaustion", detection.DetectorName)
	assert.Equal(t, models.CategoryConnection, detection.Category)
	assert.Equal(t, models.SeverityWarning, detection.Severity)
}

func TestConnectionPoolDetector_NoDetectionBelowThreshold(t *testing.T) {
	det := detector.NewConnectionPoolDetection()
	det.SetThreshold(0.8)
	active := int32(50)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire at 50pct usage.")
}

func TestConnectionPoolDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: nil,
			MaxConnections:    nil,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing.")
}

func TestConnectionPoolDetector_CriticalSeverity(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(99)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire at 99pct usage.")
	assert.Equal(t, models.SeverityCritical, detection.Severity)
}
func TestConnectionPoolDetector_InfoSeverity(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(82)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire at 82pct usage.")
	assert.Equal(t, models.SeverityInfo, detection.Severity)
}

func TestConnectionPoolDetector_PostgresRecommendation(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(92)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "PgBouncer")
	assert.Equal(t, "pgbouncer", detection.ActionMetadata["recommended_tool"])
}

func TestConnectionPoolDetector_MYSQLRecommendation(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(92)
	max := int32(100)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "mysql",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "ProxySQL")
	assert.Equal(t, "proxysql", detection.ActionMetadata["recommended_tool"])
}

func TestConnectionPoolDetector_ZeroDivisionProtection(t *testing.T) {
	det := detector.NewConnectionPoolDetection()

	active := int32(92)
	max := int32(0)

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when max is 0")
}
