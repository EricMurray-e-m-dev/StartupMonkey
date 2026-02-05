package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestTableBloatDetector_FiresWhenAboveThreshold(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.15,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 15000,
			"pg.table.posts.bloat_ratio": 0.15,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection, "Detection should fire when bloat ratio > 10%")
	assert.Equal(t, "table_bloat", detection.DetectorName)
	assert.Equal(t, models.CategoryStorage, detection.Category)
	assert.Equal(t, "vacuum_table", detection.ActionType)
}

func TestTableBloatDetector_NoDetectionWhenBelowThreshold(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.05,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 5000,
			"pg.table.posts.bloat_ratio": 0.05,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when bloat ratio < 10%")
}

func TestTableBloatDetector_NoDetectionWhenDataMissing(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:      "test-db",
		DatabaseType:    "postgres",
		Labels:          map[string]string{},
		ExtendedMetrics: map[string]float64{},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when bloat data is missing")
}

func TestTableBloatDetector_SeverityInfo(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.12,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 12000,
			"pg.table.posts.bloat_ratio": 0.12,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityInfo, detection.Severity, "10-20% bloat should be Info severity")
}

func TestTableBloatDetector_SeverityWarning(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.25,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 25000,
			"pg.table.posts.bloat_ratio": 0.25,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityWarning, detection.Severity, "20-30% bloat should be Warning severity")
}

func TestTableBloatDetector_SeverityCritical(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.35,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 35000,
			"pg.table.posts.bloat_ratio": 0.35,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, models.SeverityCritical, detection.Severity, "30%+ bloat should be Critical severity")
}

func TestTableBloatDetector_CustomThreshold(t *testing.T) {
	det := detector.NewTableBloatDetector()
	det.SetThreshold(0.2) // 20%

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.15,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 15000,
			"pg.table.posts.bloat_ratio": 0.15,
		},
	}

	detection := det.Detect(snapshot)

	assert.Nil(t, detection, "Detection should not fire when below custom 20% threshold")
}

func TestTableBloatDetector_ActionMetadata(t *testing.T) {
	det := detector.NewTableBloatDetector()

	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Labels: map[string]string{
			"pg.worst_bloat_table": "posts",
		},
		ExtendedMetrics: map[string]float64{
			"pg.worst_bloat_ratio":       0.15,
			"pg.table.posts.live_tuples": 100000,
			"pg.table.posts.dead_tuples": 15000,
			"pg.table.posts.bloat_ratio": 0.15,
		},
	}

	detection := det.Detect(snapshot)

	assert.NotNil(t, detection)
	assert.Equal(t, "posts", detection.ActionMetadata["table_name"])
	assert.Contains(t, detection.Recommendation, "VACUUM ANALYZE")
}
