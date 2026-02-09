package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestLongRunningQueryDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT * FROM large_table",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 45.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when query > 30s")
	assert.Equal(t, "long_running_query", detection.DetectorName)
	assert.Equal(t, models.CategoryQuery, detection.Category)
	assert.Equal(t, "terminate_query", detection.ActionType)
}

func TestLongRunningQueryDetector_NoDetectionWhenBelowThreshold(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT * FROM small_table",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 15.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when query < 30s")
}

func TestLongRunningQueryDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:      "test-db",
		DatabaseType:    "postgres",
		Labels:          map[string]string{},
		ExtendedMetrics: map[string]float64{},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing")
}

func TestLongRunningQueryDetector_NoPIDMissing(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels:       map[string]string{},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 60.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when PID is missing")
}

func TestLongRunningQueryDetector_SeverityInfo(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 35.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityInfo, detection.Severity, "30-60s should be Info severity")
}

func TestLongRunningQueryDetector_SeverityWarning(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 90.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityWarning, detection.Severity, "60-120s should be Warning severity")
}

func TestLongRunningQueryDetector_SeverityCritical(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 150.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityCritical, detection.Severity, "120s+ should be Critical severity")
}

func TestLongRunningQueryDetector_CustomThreshold(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()
	det.SetThreshold(60.0) // 60 seconds

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 45.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when below custom 60s threshold")
}

func TestLongRunningQueryDetector_ActionMetadata(t *testing.T) {
	det := detector.NewLongRunningQueryDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.longest_query_pid":  "12345",
			"pg.longest_query_user": "app_user",
			"pg.longest_query_text": "SELECT * FROM posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.longest_query_duration_secs": 60.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, "12345", detection.ActionMetadata["pid"])
	assert.Equal(t, "app_user", detection.ActionMetadata["username"])
	assert.Equal(t, true, detection.ActionMetadata["graceful"])
}
