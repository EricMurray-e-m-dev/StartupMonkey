package normaliser

import (
	"math"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
)

type PostgresNormaliser struct{}

func (n *PostgresNormaliser) Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error) {
	normalised := &NormalisedMetrics{
		DatabaseID:      raw.DatabaseID,
		DatabaseType:    raw.DatabaseType,
		Timestamp:       raw.Timestamp,
		Measurements:    Measurements{},
		ExtendedMetrics: make(map[string]float64),
	}

	var availableMetrics []string

	// === CONNECTION HEALTH ===
	if raw.Connections != nil && raw.Connections.Active != nil && raw.Connections.Max != nil {
		active := float64(*raw.Connections.Active)
		max := float64(*raw.Connections.Max)

		normalised.ConnectionHealth = 1.0 - (active - max)

		normalised.Measurements.ActiveConnections = raw.Connections.Active
		normalised.Measurements.MaxConnections = raw.Connections.Max
		normalised.Measurements.IdleConnections = raw.Connections.Idle
		normalised.Measurements.WaitingConnections = raw.Connections.Waiting

		availableMetrics = append(availableMetrics, "connections")
	} else {
		// Assume healthy if data not available
		normalised.ConnectionHealth = 1.0
	}

	// === QUERY HEALTH ===
	if raw.Queries != nil {

		var latency float64
		if raw.Queries.P95LatencyMs != nil {
			latency = *raw.Queries.P95LatencyMs
			normalised.Measurements.P95QueryLatencyMs = raw.Queries.P95LatencyMs
		} else if raw.Queries.AvgLatencyMs != nil {
			latency = *raw.Queries.AvgLatencyMs
		}

		if latency > 0 {
			// Health Score: 1.0 = 0ms | 0.0 = 1000ms+
			normalised.QueryHealth = math.Max(0, 1.0-(latency/1000.0))

			// Store Measurements
			normalised.Measurements.AvgQueryLatencyMs = raw.Queries.AvgLatencyMs
			normalised.Measurements.P50QueryLatencyMs = raw.Queries.P50LatencyMs
			normalised.Measurements.P99QueryLatencyMs = raw.Queries.P99LatencyMs

			if raw.Queries.SequentialScans != nil {
				normalised.Measurements.SequentialScans = raw.Queries.SequentialScans
			}

			if raw.Queries.SlowQueries != nil {
				slowCount := int32(len(raw.Queries.SlowQueries))
				normalised.Measurements.SlowQueryCount = &slowCount
			}

			availableMetrics = append(availableMetrics, "query_latency")
		} else {
			normalised.QueryHealth = 1.0
		}
	} else {
		normalised.QueryHealth = 1.0
	}

	// === STORAGE HEALTH ===
	if raw.Storage != nil && raw.Storage.UsedSizeBytes != nil && raw.Storage.TotalSizeBytes != nil {
		used := float64(*raw.Storage.UsedSizeBytes)
		total := float64(*raw.Storage.TotalSizeBytes)

		if total > 0 {
			// Health Score: 1.0 = Empty | 0.0 = Full
			normalised.StorageHealth = 1.0 - (used / total)

			// Store measurements
			normalised.Measurements.UsedStorageBytes = raw.Storage.UsedSizeBytes
			normalised.Measurements.TotalStorageBytes = raw.Storage.TotalSizeBytes
			normalised.Measurements.FreeStorageBytes = raw.Storage.FreeSpaceBytes

			availableMetrics = append(availableMetrics, "storage")
		} else {
			normalised.StorageHealth = 1.0
		}
	} else {
		normalised.StorageHealth = 1.0
	}

	// === CACHE HEALTH ===
	if raw.Cache != nil && raw.Cache.HitRate != nil {
		// Cache rate already 0.0 - 1.0
		normalised.CacheHealth = *raw.Cache.HitRate

		// Store measurements
		normalised.Measurements.CacheHitRate = raw.Cache.HitRate
		normalised.Measurements.CacheHitCount = raw.Cache.HitCount
		normalised.Measurements.CacheMissCount = raw.Cache.MissCount

		availableMetrics = append(availableMetrics, "cache")
	} else {
		normalised.CacheHealth = 1.0
	}

	// === OVERALL HEALTH ===
	if len(availableMetrics) > 0 {
		total := normalised.ConnectionHealth + normalised.QueryHealth + normalised.StorageHealth + normalised.CacheHealth
		normalised.HealthScore = total / float64(len(availableMetrics))
	} else {
		// No metrics at all
		normalised.HealthScore = 1.0
	}

	normalised.AvailableMetrics = availableMetrics

	if raw.ExtendedMetrics != nil {
		normalised.ExtendedMetrics = raw.ExtendedMetrics
	}

	return normalised, nil
}
