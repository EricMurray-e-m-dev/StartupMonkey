package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type CacheMissDetector struct {
	hitRateThreshold float64
}

func NewCacheMissDetector() *CacheMissDetector {
	return &CacheMissDetector{
		hitRateThreshold: 0.90, // Alert if hit rate falls under 90%
	}
}

func (d *CacheMissDetector) Name() string {
	return "cache_miss_rate_high"
}

func (d *CacheMissDetector) Category() models.DetectionCategory {
	return models.CategoryCache
}

func (d *CacheMissDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	if snapshot.Measurements.CacheHitRate == nil {
		return nil
	}

	hitRate := *snapshot.Measurements.CacheHitRate

	// No issues if above threshold
	if hitRate >= d.hitRateThreshold {
		return nil
	}

	var severity models.DetectionSeverity
	if hitRate < 0.70 {
		severity = models.SeverityCritical
	} else if hitRate < 0.85 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	hitPercent := int(hitRate * 100)
	missPercent := 100 - hitPercent

	detection.Title = fmt.Sprintf("Cache hit rate at %d%% (%d%% miss rate)", hitPercent, missPercent)
	detection.Description = fmt.Sprintf(
		"Database cache hit rate is only %.1f%%, meaning %.1f%% of reads require disk I/O. "+
			"Low cache hit rates significantly degrade query performance, especially under load. "+
			"This typically indicates insufficient memory allocated to database cache.",
		hitRate*100, (1-hitRate)*100,
	)

	detection.Evidence = map[string]interface{}{
		"cache_hit_rate":     hitRate,
		"cache_hit_percent":  hitPercent,
		"cache_miss_percent": missPercent,
		"cache_health":       snapshot.CacheHealth,
	}

	detection.Recommendation = d.getRecommendation(snapshot.DatabaseType, hitRate)

	detection.ActionType = "increase_cache_size"
	detection.ActionMetadata = map[string]interface{}{
		"priority":         "medium",
		"database_type":    snapshot.DatabaseType,
		"current_hit_rate": hitPercent,
		"target_hit_rate":  95,
	}

	return detection
}

func (d *CacheMissDetector) getRecommendation(dbType string, hitRate float64) string {
	switch dbType {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"Increase PostgreSQL's shared_buffers setting to allocate more memory for caching. "+
				"Current cache hit rate: %.1f%%. Recommended: Increase shared_buffers from current value "+
				"(typically 25%% of system RAM is a good starting point). "+
				"Restart required after changing shared_buffers.",
			hitRate*100,
		)
	case "mysql":
		return fmt.Sprintf(
			"Increase MySQL's innodb_buffer_pool_size to allocate more memory for caching. "+
				"Current cache hit rate: %.1f%%. Recommended: Set innodb_buffer_pool_size to 70-80%% "+
				"of available system RAM for dedicated database servers.",
			hitRate*100,
		)
	case "mongodb":
		return fmt.Sprintf(
			"MongoDB uses the operating system's file system cache (WiredTiger cache). "+
				"Current cache hit rate: %.1f%%. Recommended: Increase available system memory "+
				"or reduce working set size. MongoDB typically uses 50%% of RAM minus 1GB for cache.",
			hitRate*100,
		)
	case "sqlite":
		return fmt.Sprintf(
			"SQLite cache performance is low (%.1f%% hit rate). "+
				"Increase PRAGMA cache_size to allocate more memory for page cache. "+
				"Note: SQLite cache is limited compared to server databases like PostgreSQL.",
			hitRate*100,
		)
	default:
		return fmt.Sprintf(
			"Cache hit rate is low (%.1f%%). Increase database cache/buffer pool size "+
				"to improve performance. Consult your database documentation for cache configuration.",
			hitRate*100,
		)
	}
}

func (d *CacheMissDetector) SetThreshold(threshold float64) {
	d.hitRateThreshold = threshold
}
