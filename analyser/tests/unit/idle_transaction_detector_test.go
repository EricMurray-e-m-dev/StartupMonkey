package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestIdleTransactionDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT * FROM users WHERE id = 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 360.0, // 6 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when idle > 5 minutes")
	assert.Equal(t, "idle_transaction", detection.DetectorName)
	assert.Equal(t, models.CategoryConnection, detection.Category)
	assert.Equal(t, "terminate_query", detection.ActionType)
}

func TestIdleTransactionDetector_NoDetectionWhenBelowThreshold(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 120.0, // 2 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when idle < 5 minutes")
}

func TestIdleTransactionDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:      "test-db",
		DatabaseType:    "postgres",
		Labels:          map[string]string{},
		ExtendedMetrics: map[string]float64{},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing")
}

func TestIdleTransactionDetector_NoPIDMissing(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels:       map[string]string{},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 600.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when PID is missing")
}

func TestIdleTransactionDetector_SeverityInfo(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 360.0, // 6 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityInfo, detection.Severity, "5-10 minutes should be Info severity")
}

func TestIdleTransactionDetector_SeverityWarning(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 720.0, // 12 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityWarning, detection.Severity, "10-15 minutes should be Warning severity")
}

func TestIdleTransactionDetector_SeverityCritical(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 1200.0, // 20 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityCritical, detection.Severity, "15+ minutes should be Critical severity")
}

func TestIdleTransactionDetector_CustomThreshold(t *testing.T) {
	det := detector.NewIdleTransactionDetector()
	det.SetThreshold(600.0) // 10 minutes

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 480.0, // 8 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when below custom 10 minute threshold")
}

func TestIdleTransactionDetector_ActionMetadata(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "BEGIN; SELECT * FROM users",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 400.0,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, "12345", detection.ActionMetadata["pid"])
	assert.Equal(t, "app_user", detection.ActionMetadata["username"])
	assert.Equal(t, false, detection.ActionMetadata["graceful"], "Idle transactions should use forceful termination")
}

func TestIdleTransactionDetector_TitleShowsMinutes(t *testing.T) {
	det := detector.NewIdleTransactionDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.idle_txn_pid":   "12345",
			"pg.idle_txn_user":  "app_user",
			"pg.idle_txn_query": "SELECT 1",
		},
		ExtendedMetrics: map[string]float64{
			"pg.idle_txn_duration_secs": 600.0, // 10 minutes
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Title, "10 minutes")
}
