package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type HighLatencyDetector struct {
	p95LatencyThreshold float64
}

func NewHighLatencyDetector() *HighLatencyDetector {
	return &HighLatencyDetector{
		p95LatencyThreshold: 100.0, // Alert if p95 over 500ms
	}
}

func (d *HighLatencyDetector) Name() string {
	return "high_query_latency"
}

func (d *HighLatencyDetector) Category() models.DetectionCategory {
	return models.CategoryQuery
}

func (d *HighLatencyDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	// Prefer p95 latency, fallback to average
	var latency float64
	var latencyType string

	if snapshot.Measurements.P95QueryLatencyMs != nil {
		latency = *snapshot.Measurements.P95QueryLatencyMs
		latencyType = "p95"
	} else if snapshot.Measurements.AvgQueryLatencyMs != nil {
		latency = *snapshot.Measurements.AvgQueryLatencyMs
		latencyType = "avg"
	} else {
		return nil // No latency metrics available
	}

	// No issues if below threshold
	if latency < d.p95LatencyThreshold {
		return nil
	}

	var severity models.DetectionSeverity
	if latency > d.p95LatencyThreshold*3 {
		severity = models.SeverityCritical // If over 3x threshold
	} else if latency > d.p95LatencyThreshold*2 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	detection.Title = fmt.Sprintf("High query latency detected (%.0fms %s)", latency, latencyType)
	detection.Description = fmt.Sprintf(
		"Database queries have high execution times (%s: %.0fms, threshold: %.0fms). "+
			"Slow queries degrade application responsiveness and user experience. "+
			"Common causes include missing indexes, inefficient queries, insufficient memory allocation, or suboptimal configuration.",
		latencyType, latency, d.p95LatencyThreshold,
	)

	evidence := map[string]interface{}{
		"latency_ms":   latency,
		"latency_type": latencyType,
		"threshold_ms": d.p95LatencyThreshold,
		"query_health": snapshot.QueryHealth,
	}

	// Include all available latency metrics
	if snapshot.Measurements.AvgQueryLatencyMs != nil {
		evidence["avg_latency_ms"] = *snapshot.Measurements.AvgQueryLatencyMs
	}
	if snapshot.Measurements.P50QueryLatencyMs != nil {
		evidence["p50_latency_ms"] = *snapshot.Measurements.P50QueryLatencyMs
	}
	if snapshot.Measurements.P95QueryLatencyMs != nil {
		evidence["p95_latency_ms"] = *snapshot.Measurements.P95QueryLatencyMs
	}
	if snapshot.Measurements.P99QueryLatencyMs != nil {
		evidence["p99_latency_ms"] = *snapshot.Measurements.P99QueryLatencyMs
	}

	detection.Evidence = evidence

	detection.Recommendation = d.getRecommendation(snapshot.DatabaseType)

	// NEW: Use tune_config_high_latency action instead of optimise_queries
	detection.ActionType = "tune_config_high_latency"
	detection.ActionMetadata = map[string]interface{}{
		"database_type":  snapshot.DatabaseType,
		"p95_latency_ms": latency,
		"threshold_ms":   d.p95LatencyThreshold,
	}

	// Include avg latency if available
	if snapshot.Measurements.AvgQueryLatencyMs != nil {
		detection.ActionMetadata["avg_latency_ms"] = *snapshot.Measurements.AvgQueryLatencyMs
	}

	return detection
}

func (d *HighLatencyDetector) getRecommendation(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		return "StartupMonkey will tune PostgreSQL configuration (work_mem, effective_cache_size, random_page_cost) " +
			"to improve query performance and identify slow queries that require code changes."
	case "mysql":
		return "StartupMonkey will tune MySQL configuration (innodb_buffer_pool_size, tmp_table_size) " +
			"to improve query performance and identify slow queries that require optimization."
	case "mongodb":
		return "StartupMonkey will tune MongoDB configuration (wiredTigerCacheSizeGB) " +
			"to improve query performance and identify slow queries that require optimization."
	case "sqlite":
		return "StartupMonkey will tune SQLite configuration (cache_size, journal_mode) " +
			"to improve query performance. For high-traffic applications, consider migrating to PostgreSQL or MySQL."
	default:
		return "StartupMonkey will optimize database configuration to improve query performance."
	}
}

func (d *HighLatencyDetector) SetThreshold(threshold float64) {
	d.p95LatencyThreshold = threshold
}
