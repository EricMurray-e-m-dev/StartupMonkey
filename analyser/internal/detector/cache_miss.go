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

	// Changed from "increase_cache_size" to "cache_optimization_recommendation"
	detection.ActionType = "cache_optimization_recommendation"
	detection.ActionMetadata = map[string]interface{}{
		"priority":         "medium",
		"database_type":    snapshot.DatabaseType,
		"current_hit_rate": hitPercent,
		"target_hit_rate":  95,

		// Safe option: Increase database cache
		"safe_option": map[string]interface{}{
			"title":            d.getSafeOptionTitle(snapshot.DatabaseType),
			"description":      d.getSafeOptionDescription(snapshot.DatabaseType, hitRate),
			"risk_level":       "safe",
			"requires_restart": true,
			"steps":            d.getSafeOptionSteps(snapshot.DatabaseType),
		},

		// Advanced option: Deploy Redis
		"advanced_option": map[string]interface{}{
			"title":             "Deploy Redis Cache Layer",
			"description":       d.getAdvancedOptionDescription(snapshot.DatabaseType, hitRate),
			"risk_level":        "advanced",
			"requires_restart":  false,
			"deployable_action": "deploy_redis",
			"warning": "Requires modifying application query logic. " +
				"Not recommended for beginners. Test thoroughly before production deployment.",
		},
	}

	return detection
}

// getSafeOptionTitle returns database-specific title for safe cache increase
func (d *CacheMissDetector) getSafeOptionTitle(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		return "Increase PostgreSQL shared_buffers"
	case "mysql":
		return "Increase MySQL InnoDB Buffer Pool"
	case "mongodb":
		return "Increase MongoDB WiredTiger Cache"
	case "sqlite":
		return "Increase SQLite Cache Size"
	default:
		return "Increase Database Cache"
	}
}

// getSafeOptionDescription returns database-specific description for safe option
func (d *CacheMissDetector) getSafeOptionDescription(dbType string, hitRate float64) string {
	switch dbType {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"Increase shared_buffers to improve cache hit rate from %.1f%% to 95%%+. "+
				"Recommended: 25%% of system RAM.",
			hitRate*100,
		)
	case "mysql":
		return fmt.Sprintf(
			"Increase innodb_buffer_pool_size to improve cache hit rate from %.1f%% to 95%%+. "+
				"Recommended: 70-80%% of system RAM for dedicated servers.",
			hitRate*100,
		)
	case "mongodb":
		return fmt.Sprintf(
			"Increase WiredTiger cache to improve cache hit rate from %.1f%% to 95%%+. "+
				"MongoDB typically uses 50%% of RAM minus 1GB.",
			hitRate*100,
		)
	case "sqlite":
		return fmt.Sprintf(
			"Increase PRAGMA cache_size to improve cache hit rate from %.1f%% to 95%%+. "+
				"Note: SQLite cache is limited compared to server databases.",
			hitRate*100,
		)
	default:
		return fmt.Sprintf(
			"Increase database cache size to improve cache hit rate from %.1f%% to 95%%+.",
			hitRate*100,
		)
	}
}

// getSafeOptionSteps returns database-specific steps for safe cache increase
func (d *CacheMissDetector) getSafeOptionSteps(dbType string) []string {
	switch dbType {
	case "postgres", "postgresql":
		return []string{
			"Locate postgresql.conf file",
			"Find the shared_buffers setting",
			"Increase to 256MB (or 25% of system RAM)",
			"Save the file",
			"Restart PostgreSQL service",
			"Monitor cache hit rate in Dashboard",
		}
	case "mysql":
		return []string{
			"Locate my.cnf or my.ini file",
			"Find the innodb_buffer_pool_size setting",
			"Increase to 512MB (or 70% of system RAM)",
			"Save the file",
			"Restart MySQL service",
			"Monitor cache hit rate in Dashboard",
		}
	case "mongodb":
		return []string{
			"Locate mongod.conf file",
			"Find the wiredTigerCacheSizeGB setting",
			"Increase to appropriate size (50% RAM - 1GB)",
			"Save the file",
			"Restart MongoDB service",
			"Monitor cache hit rate in Dashboard",
		}
	case "sqlite":
		return []string{
			"Add PRAGMA cache_size command to connection initialization",
			"Set cache_size to 10000 or higher",
			"Restart application",
			"Monitor query performance",
		}
	default:
		return []string{
			"Review database documentation for cache configuration",
			"Identify cache-related setting",
			"Increase cache size appropriately",
			"Restart database service",
			"Monitor performance",
		}
	}
}

// getAdvancedOptionDescription returns description for Redis deployment option
func (d *CacheMissDetector) getAdvancedOptionDescription(dbType string, hitRate float64) string {
	return fmt.Sprintf(
		"Deploy Redis as an application-level cache layer to improve cache hit rate from %.1f%% to 95%%+. "+
			"This approach requires modifying your application code to query Redis before the database. "+
			"Provides maximum performance gains but requires development effort and testing.",
		hitRate*100,
	)
}

// getRecommendation returns general recommendation text (kept for backwards compatibility)
func (d *CacheMissDetector) getRecommendation(dbType string, hitRate float64) string {
	switch dbType {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"Cache hit rate is low (%.1f%%). Two options: "+
				"1) Increase shared_buffers in postgresql.conf (requires restart), or "+
				"2) Deploy Redis for application-level caching (requires code changes).",
			hitRate*100,
		)
	case "mysql":
		return fmt.Sprintf(
			"Cache hit rate is low (%.1f%%). Two options: "+
				"1) Increase innodb_buffer_pool_size in my.cnf (requires restart), or "+
				"2) Deploy Redis for application-level caching (requires code changes).",
			hitRate*100,
		)
	case "mongodb":
		return fmt.Sprintf(
			"Cache hit rate is low (%.1f%%). Two options: "+
				"1) Increase wiredTigerCacheSizeGB (requires restart), or "+
				"2) Deploy Redis for application-level caching (requires code changes).",
			hitRate*100,
		)
	default:
		return fmt.Sprintf(
			"Cache hit rate is low (%.1f%%). Review database cache configuration "+
				"or consider deploying Redis for application-level caching.",
			hitRate*100,
		)
	}
}

func (d *CacheMissDetector) SetThreshold(threshold float64) {
	d.hitRateThreshold = threshold
}
