package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestMissingIndexDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewMissingIndexDetector()

	seqScans := int32(15)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: &seqScans,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when sequential scans are above 10")
	assert.Equal(t, "missing_index", detection.DetectorName)
	assert.Equal(t, models.CategoryQuery, detection.Category)
	assert.Equal(t, models.SeverityWarning, detection.Severity)
}
func TestMissingIndexDetector_NoDetectionWhenBelowThreshold(t *testing.T) {
	det := detector.NewMissingIndexDetector()

	seqScans := int32(5)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: &seqScans,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when sequential scans are below 10")
}

func TestMissingIndexDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewMissingIndexDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: nil,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when data is missing")
}

func TestMissingIndexDetector_ExactlyAtThreshold(t *testing.T) {
	det := detector.NewMissingIndexDetector()

	seqScans := int32(10)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: &seqScans,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when at threshold")
}

func TestMissingIndexDetector_CustomThreshold(t *testing.T) {
	det := detector.NewMissingIndexDetector()
	det.SetThreshold(5)

	seqScans := int32(10)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: &seqScans,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when above new custom threshold")
}

func TestMissingIndexDetector_RecommendationContent(t *testing.T) {
	det := detector.NewMissingIndexDetector()

	seqScans := int32(20)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			SequentialScans: &seqScans,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Contains(t, detection.Recommendation, "CREATE INDEX")
	assert.Contains(t, detection.Recommendation, "CONCURRENTLY")
	assert.Equal(t, "create_index", detection.ActionType)
}
