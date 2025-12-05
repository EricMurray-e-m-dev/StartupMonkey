package normaliser

import (
	"math"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
)

type PostgresNormaliser struct {
	previousMetrics map[string]*NormalisedMetrics
}

func NewPostgresNormaliser() *PostgresNormaliser {
	return &PostgresNormaliser{
		previousMetrics: make(map[string]*NormalisedMetrics),
	}
}

func (n *PostgresNormaliser) Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error) {
	normalised := &NormalisedMetrics{
		DatabaseID:       raw.DatabaseID,
		DatabaseType:     raw.DatabaseType,
		Timestamp:        raw.Timestamp,
		Measurements:     Measurements{},
		MetricDeltas:     map[string]float64{},
		TimeDeltaSeconds: 0,
		ExtendedMetrics:  make(map[string]float64),
		Labels:           make(map[string]string),
	}

	var healthScores []float64 // Track only scores for available metrics

	// === CONNECTION HEALTH ===
	if raw.Connections != nil && raw.Connections.Active != nil && raw.Connections.Max != nil {
		active := float64(*raw.Connections.Active)
		max := float64(*raw.Connections.Max)

		if max > 0 {
			normalised.ConnectionHealth = 1.0 - (active / max)
		} else {
			normalised.ConnectionHealth = 1.0
		}

		normalised.Measurements.ActiveConnections = raw.Connections.Active
		normalised.Measurements.MaxConnections = raw.Connections.Max
		normalised.Measurements.IdleConnections = raw.Connections.Idle
		normalised.Measurements.WaitingConnections = raw.Connections.Waiting

		healthScores = append(healthScores, normalised.ConnectionHealth)
	} else {
		normalised.ConnectionHealth = 1.0
	}

	// === QUERY HEALTH ===
	if raw.Queries != nil {
		var queryHealth float64 = 1.0
		hasQueryMetrics := false

		// Map sequential scans
		if raw.Queries.SequentialScans != nil {
			normalised.Measurements.SequentialScans = raw.Queries.SequentialScans
			hasQueryMetrics = true
		}

		// Latency-based health
		var latency float64
		if raw.Queries.P95LatencyMs != nil {
			latency = *raw.Queries.P95LatencyMs
			normalised.Measurements.P95QueryLatencyMs = raw.Queries.P95LatencyMs
		} else if raw.Queries.AvgLatencyMs != nil {
			latency = *raw.Queries.AvgLatencyMs
		}

		if latency > 0 {
			// Health Score: 1.0 = 0ms | 0.5 = 500ms | 0.0 = 1000ms+
			queryHealth = math.Max(0, 1.0-(latency/1000.0))

			normalised.Measurements.AvgQueryLatencyMs = raw.Queries.AvgLatencyMs
			normalised.Measurements.P50QueryLatencyMs = raw.Queries.P50LatencyMs
			normalised.Measurements.P99QueryLatencyMs = raw.Queries.P99LatencyMs

			if raw.Queries.SlowQueries != nil {
				slowCount := int32(len(raw.Queries.SlowQueries))
				normalised.Measurements.SlowQueryCount = &slowCount
			}

			hasQueryMetrics = true
		}

		// Penalize for sequential scans (they indicate missing indexes)
		if raw.Queries.SequentialScans != nil {
			seqScans := float64(*raw.Queries.SequentialScans)
			// Reduce health by 10% for every 100 sequential scans (capped at 50% reduction)
			seqScanPenalty := math.Min(0.5, (seqScans/100.0)*0.1)
			queryHealth = math.Max(0, queryHealth-seqScanPenalty)
		}

		normalised.QueryHealth = queryHealth

		if hasQueryMetrics {
			healthScores = append(healthScores, normalised.QueryHealth)
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

			normalised.Measurements.UsedStorageBytes = raw.Storage.UsedSizeBytes
			normalised.Measurements.TotalStorageBytes = raw.Storage.TotalSizeBytes
			normalised.Measurements.FreeStorageBytes = raw.Storage.FreeSpaceBytes

			healthScores = append(healthScores, normalised.StorageHealth)
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

		normalised.Measurements.CacheHitRate = raw.Cache.HitRate
		normalised.Measurements.CacheHitCount = raw.Cache.HitCount
		normalised.Measurements.CacheMissCount = raw.Cache.MissCount

		healthScores = append(healthScores, normalised.CacheHealth)
	} else {
		normalised.CacheHealth = 1.0
	}

	// === OVERALL HEALTH (FIXED) ===
	if len(healthScores) > 0 {
		var total float64
		for _, score := range healthScores {
			total += score
		}
		normalised.HealthScore = total / float64(len(healthScores))
	} else {
		normalised.HealthScore = 1.0
	}

	if raw.ExtendedMetrics != nil {
		normalised.ExtendedMetrics = raw.ExtendedMetrics
	}

	if raw.Labels != nil {
		normalised.Labels = raw.Labels
	}

	n.calculateDeltas(normalised)

	n.previousMetrics[normalised.DatabaseID] = normalised

	return normalised, nil
}

func (n *PostgresNormaliser) calculateDeltas(current *NormalisedMetrics) {
	previous, exists := n.previousMetrics[current.DatabaseID]

	if !exists {
		current.TimeDeltaSeconds = 0
		current.MetricDeltas = make(map[string]float64)
		return
	}

	timeDelta := float64(current.Timestamp - previous.Timestamp)
	if timeDelta <= 0 {
		current.TimeDeltaSeconds = 0
		return
	}
	current.TimeDeltaSeconds = timeDelta

	// FIXED: Sequential scans delta (was comparing current to current)
	if current.Measurements.SequentialScans != nil && previous.Measurements.SequentialScans != nil {
		currentVal := float64(*current.Measurements.SequentialScans)
		previousVal := float64(*previous.Measurements.SequentialScans) // Fixed!
		delta := currentVal - previousVal

		if delta < 0 {
			delta = 0
		}

		current.MetricDeltas["sequential_scans"] = delta
	}

	// Slow query count delta
	if current.Measurements.SlowQueryCount != nil && previous.Measurements.SlowQueryCount != nil {
		currentVal := float64(*current.Measurements.SlowQueryCount)
		previousVal := float64(*previous.Measurements.SlowQueryCount)
		delta := currentVal - previousVal

		if delta < 0 {
			delta = 0
		}

		current.MetricDeltas["slow_query_count"] = delta
	}

	// Cache miss count delta
	if current.Measurements.CacheMissCount != nil && previous.Measurements.CacheMissCount != nil {
		currentVal := float64(*current.Measurements.CacheMissCount)
		previousVal := float64(*previous.Measurements.CacheMissCount)
		delta := currentVal - previousVal

		if delta < 0 {
			delta = 0
		}

		current.MetricDeltas["cache_miss_count"] = delta
	}
}
