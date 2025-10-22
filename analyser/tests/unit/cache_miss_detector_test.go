package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestCacheRateMissDetector_FiresWhenBelowThreshold(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 0.85
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Timestamp:    123456,
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when hit rate < 90pct")
	assert.Equal(t, "cache_miss_rate_high", detection.DetectorName)
	assert.Equal(t, models.CategoryCache, detection.Category)
	assert.Equal(t, models.SeverityInfo, detection.Severity)
	assert.Contains(t, detection.Title, "85%")
}

func TestCacheRateMissDetector_NoDetectionWhenAboveThreshold(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 1.00
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when > 90pct")
}

func TestCacheRateMissDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewCacheMissDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: nil,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing")
}

func TestCacheRateMissDetector_CriticalSeverity(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 0.1
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityCritical, detection.Severity)
}

func TestCacheRateMissDetector_WarningSeverity(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 0.80
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityWarning, detection.Severity)
}

func TestCacheMissDetector_PostgresRecommendation(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 0.85
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "shared_buffers")
	assert.Contains(t, detection.Recommendation, "PostgreSQL")
}

func TestCacheMissDetector_MySQLRecommendation(t *testing.T) {
	det := detector.NewCacheMissDetector()

	hitRate := 0.85
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "mysql",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "innodb_buffer_pool_size")
	assert.Contains(t, detection.Recommendation, "MySQL")
}

func TestCacheMissDetector_CustomThreshold(t *testing.T) {
	det := detector.NewCacheMissDetector()
	det.SetThreshold(0.95) // Raise threshold to 95%

	hitRate := 0.92 // 92% (would normally pass, but below new threshold)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire with custom threshold")
}
