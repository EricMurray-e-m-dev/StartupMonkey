package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type ConnectionPoolDetector struct {
	usageThreshold float64
}

func NewConnectionPoolDetection() *ConnectionPoolDetector {
	return &ConnectionPoolDetector{
		usageThreshold: 0.8,
	}
}

func (d *ConnectionPoolDetector) Name() string {
	return "connection_pool_exhaustion"
}

func (d *ConnectionPoolDetector) Category() models.DetectionCategory {
	return models.CategoryConnection
}

func (d *ConnectionPoolDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	if snapshot.Measurements.ActiveConnections == nil || snapshot.Measurements.MaxConnections == nil {
		return nil // Cant calculate pool cap without data
	}

	active := float64(*snapshot.Measurements.ActiveConnections)
	max := float64(*snapshot.Measurements.MaxConnections)

	if max == 0 {
		return nil
	}

	usageRatio := active / max

	// Check if below threshold
	if usageRatio < d.usageThreshold {
		return nil
	}

	var severity models.DetectionSeverity

	if usageRatio >= 0.95 {
		severity = models.SeverityCritical
	} else if usageRatio >= 0.85 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	usagePercentage := int(usageRatio * 100)
	detection.Title = fmt.Sprintf("Connection pool at %d%% capacity", usagePercentage)
	detection.Description = fmt.Sprintf(
		"Database connection pool is using %d out of %d available connections (%.1f%%) "+
			"When the pool is exhausted, new connections will be queued or refused. "+
			"Causing timeouts & degrading user experience",
		int(active), int(max), usageRatio*100,
	)

	detection.Evidence = map[string]interface{}{
		"active_connections": int(active),
		"max_connections":    int(max),
		"usage_ratio":        usageRatio,
		"usage_percent":      usagePercentage,
		"connection_health":  snapshot.ConnectionHealth,
	}

	detection.Recommendation = d.getRecommendation(snapshot.DatabaseType, usagePercentage)
	// For Executor
	detection.ActionType = "deploy_connection_pooler"
	detection.ActionMetadata = map[string]interface{}{
		"priority":         "high",
		"database_type":    snapshot.DatabaseType,
		"recommended_tool": d.getRecommendedTool(snapshot.DatabaseType),
		"current_usage":    usagePercentage,
	}

	return detection
}

func (d *ConnectionPoolDetector) getRecommendation(dbType string, usagePercent int) string {
	switch dbType {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"Deploy PgBouncer to manage PostgreSQL connections efficiently. "+
				"PgBouncer reduces connection overhead by pooling and reusing connections. "+
				"Current usage: %d%%. Recommended: PgBouncer with pool_size=%d",
			usagePercent, d.calculateRecommendedPoolSize(usagePercent),
		)
	case "mysql":
		return fmt.Sprintf(
			"Deploy ProxySQL to manage MySQL connections efficiently. "+
				"ProxySQL provides connection pooling and query routing. "+
				"Current usage: %d%%.",
			usagePercent,
		)
	case "mongodb":
		return "MongoDB drivers include built-in connection pooling. " +
			"Increase maxPoolSize in your connection string or driver configuration."
	case "sqlite":
		return "SQLite uses a single-writer model and doesn't support connection pooling. " +
			"Consider migrating to PostgreSQL or MySQL for better concurrency."
	default:
		return "Connection pool exhaustion detected. Deploy a connection pooler appropriate for your database."
	}
}

func (d *ConnectionPoolDetector) getRecommendedTool(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		return "pgbouncer"
	case "mysql":
		return "proxysql"
	case "mongodb":
		return "driver_config"
	default:
		return "unknown"
	}
}

func (d *ConnectionPoolDetector) calculateRecommendedPoolSize(usagePercent int) int {
	if usagePercent > 80 {
		return 200
	}

	return 100
}

func (d *ConnectionPoolDetector) SetThreshold(threshold float64) {
	d.usageThreshold = threshold
}
