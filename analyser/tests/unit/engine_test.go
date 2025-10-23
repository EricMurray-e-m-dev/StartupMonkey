package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/stretchr/testify/assert"
)

func TestEngine_RegisterDetector(t *testing.T) {
	eng := engine.NewEngine()

	det := detector.NewCacheMissDetector()
	eng.RegisterDetector(det)

	detectors := eng.GetRegisteredDetectors()

	assert.Len(t, detectors, 1)
	assert.Contains(t, detectors, "cache_miss_rate_high")
}

func TestEngine_RunDetectors_NoIssues(t *testing.T) {
	eng := engine.NewEngine()
	eng.RegisterDetector(detector.NewCacheMissDetector())

	hitRate := 0.95
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			CacheHitRate: &hitRate,
		},
	}

	detections := eng.RunDetectors(snapshot)

	assert.Empty(t, detections, "No detections should fire when DB is healthy")
}

func TestEngine_RunDetectors_MultipleIssues(t *testing.T) {
	eng := engine.NewEngine()
	eng.RegisterDetector(detector.NewCacheMissDetector())
	eng.RegisterDetector(detector.NewConnectionPoolDetection())

	hitRate := 0.85
	active := int32(90)
	max := int32(100)
	snapshot := &normaliser.NormalisedMetrics{
		DatabaseID:   "test-db",
		DatabaseType: "postgres",
		Measurements: normaliser.Measurements{
			ActiveConnections: &active,
			MaxConnections:    &max,
			CacheHitRate:      &hitRate,
		},
	}

	detections := eng.RunDetectors(snapshot)

	assert.Len(t, detections, 2, "Both detectors should fire")

	detectorNames := make([]string, len(detections))
	for i, d := range detections {
		detectorNames[i] = d.DetectorName
	}

	assert.Contains(t, detectorNames, "cache_miss_rate_high")
	assert.Contains(t, detectorNames, "connection_pool_exhaustion")
}
