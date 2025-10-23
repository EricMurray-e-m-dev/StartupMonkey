package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestHighLatencyDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 150.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when latency is above 100ms")
	assert.Equal(t, "high_query_latency", detection.DetectorName)
	assert.Equal(t, models.CategoryQuery, detection.Category)
	assert.Equal(t, models.SeverityInfo, detection.Severity)
}

func TestHighLatencyDetector_NoDetectionWhenBelowThreshold(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 50.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when latency is 50ms")
}

func TestHighLatencyDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: nil,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing")
}

func TestHighLatencyDetector_CriticalSeverity(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 1000.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityCritical, detection.Severity)
}

func TestHighLatencyDetector_WarningSeverity(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 250.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityWarning, detection.Severity)
}

func TestHighLatencyDetector_PostgresRecommendation(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 150.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "pg_stat_statements")
	assert.Contains(t, detection.Recommendation, "EXPLAIN ANALYZE")
}

func TestHighLatencyDetector_MySQLRecommendation(t *testing.T) {
	det := detector.NewHighLatencyDetector()

	avgLatency := 150.0
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "mysql",
		Measurements: normaliser.Measurements{
			AvgQueryLatencyMs: &avgLatency,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "slow query log")
}
