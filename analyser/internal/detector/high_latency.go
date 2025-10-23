package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type HighLatencyDetector struct {
	avgLatencyThreshold float64
}

func NewHighLatencyDetector() *HighLatencyDetector {
	return &HighLatencyDetector{
		avgLatencyThreshold: 100.0, // Alert if over 100ms
	}
}

func (d *HighLatencyDetector) Name() string {
	return "high_query_latency"
}

func (d *HighLatencyDetector) Category() models.DetectionCategory {
	return models.CategoryQuery
}

func (d *HighLatencyDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	if snapshot.Measurements.AvgQueryLatencyMs == nil {
		return nil
	}

	avgLatency := *snapshot.Measurements.AvgQueryLatencyMs

	// No issues if below threshold
	if avgLatency < d.avgLatencyThreshold {
		return nil
	}

	var severity models.DetectionSeverity
	if avgLatency > d.avgLatencyThreshold*3 {
		severity = models.SeverityCritical // If over 3x Threshold
	} else if avgLatency > d.avgLatencyThreshold*2 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	detection.Title = fmt.Sprintf("High average query latency (%.0fms)", avgLatency)
	detection.Description = fmt.Sprintf(
		"Database queries have an average execution time of %.0fms (threshold: %.0fms). "+
			"Slow queries degrade application responsiveness and user experience. "+
			"Common causes include missing indexes, inefficient queries, lock contention, or insufficient resources.",
		avgLatency, d.avgLatencyThreshold,
	)

	evidence := map[string]interface{}{
		"avg_latency_ms": avgLatency,
		"threshold_ms":   d.avgLatencyThreshold,
		"query_health":   snapshot.QueryHealth,
	}

	// Check if available (TODO: Calculate percentiles in future sprint)
	if snapshot.Measurements.P95QueryLatencyMs != nil {
		evidence["p95_latency_ms"] = *snapshot.Measurements.P95QueryLatencyMs
	}
	if snapshot.Measurements.P99QueryLatencyMs != nil {
		evidence["p99_latency_ms"] = *snapshot.Measurements.P99QueryLatencyMs
	}

	detection.Evidence = evidence

	detection.Recommendation = d.getRecommendation(snapshot.DatabaseType, avgLatency)

	// For Executor
	detection.ActionType = "optimise_queries"
	detection.ActionMetadata = map[string]interface{}{
		"priority":           "high",
		"database_type":      snapshot.DatabaseType,
		"current_latency_ms": avgLatency,
		"require_analysis":   true,
	}

	return detection
}

func (d *HighLatencyDetector) getRecommendation(dbType string, latency float64) string {
	switch dbType {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"Average query latency is high (%.0fms). To identify and fix slow queries:\n"+
				"1. Enable pg_stat_statements: CREATE EXTENSION pg_stat_statements;\n"+
				"2. Find slow queries: SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;\n"+
				"3. Run EXPLAIN ANALYZE on slow queries to identify missing indexes or inefficient plans\n"+
				"4. Check for table bloat: SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables;\n"+
				"5. Consider VACUUM ANALYZE to update statistics",
			latency,
		)
	case "mysql":
		return fmt.Sprintf(
			"Average query latency is high (%.0fms). To identify and fix slow queries:\n"+
				"1. Enable slow query log: SET GLOBAL slow_query_log = 'ON'; SET GLOBAL long_query_time = 0.1;\n"+
				"2. Review /var/log/mysql/slow-queries.log for slow queries\n"+
				"3. Run EXPLAIN on slow queries to identify missing indexes\n"+
				"4. Update table statistics: ANALYZE TABLE table_name;\n"+
				"5. Check for table locks: SHOW ENGINE INNODB STATUS;",
			latency,
		)
	case "mongodb":
		return fmt.Sprintf(
			"Average query latency is high (%.0fms). To identify and fix slow queries:\n"+
				"1. Enable profiler: db.setProfilingLevel(1, {slowms: 100});\n"+
				"2. Review slow queries: db.system.profile.find().sort({ts:-1}).limit(10);\n"+
				"3. Use explain() to analyze query plans: db.collection.find({...}).explain('executionStats');\n"+
				"4. Create indexes on frequently queried fields\n"+
				"5. Consider sharding for large collections",
			latency,
		)
	case "sqlite":
		return fmt.Sprintf(
			"Average query latency is high (%.0fms). SQLite optimization options:\n"+
				"1. Create indexes on frequently queried columns: CREATE INDEX idx_name ON table(column);\n"+
				"2. Analyze query plans: EXPLAIN QUERY PLAN SELECT ...;\n"+
				"3. Increase cache size: PRAGMA cache_size = 10000;\n"+
				"4. Enable WAL mode for better concurrency: PRAGMA journal_mode = WAL;\n"+
				"5. For high-traffic apps, consider migrating to PostgreSQL or MySQL",
			latency,
		)
	default:
		return fmt.Sprintf(
			"Average query latency is high (%.0fms). General optimization steps:\n"+
				"1. Identify slow queries using database profiling tools\n"+
				"2. Analyze query execution plans\n"+
				"3. Create appropriate indexes on filtered/joined columns\n"+
				"4. Update database statistics\n"+
				"5. Check for resource constraints (CPU, memory, disk I/O)",
			latency,
		)
	}
}

func (d *HighLatencyDetector) SetThreshold(threshold float64) {
	d.avgLatencyThreshold = threshold
}
