// Package normaliser converts raw database metrics into normalised health scores.
package normaliser

import (
	"math"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
)

// MySQLNormaliser converts raw MySQL metrics to normalised health scores.
type MySQLNormaliser struct {
	previousMetrics map[string]*NormalisedMetrics
}

// NewMySQLNormaliser creates a new MySQL normaliser.
func NewMySQLNormaliser() *MySQLNormaliser {
	return &MySQLNormaliser{
		previousMetrics: make(map[string]*NormalisedMetrics),
	}
}

// Normalise converts raw MySQL metrics to normalised health scores.
func (n *MySQLNormaliser) Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error) {
	normalised := &NormalisedMetrics{
		DatabaseID:       raw.DatabaseID,
		DatabaseType:     raw.DatabaseType,
		Timestamp:        raw.Timestamp,
		Measurements:     Measurements{},
		MetricDeltas:     make(map[string]float64),
		TimeDeltaSeconds: 0,
		ExtendedMetrics:  make(map[string]float64),
		Labels:           make(map[string]string),
	}

	var healthScores []float64

	// Connection health: based on active/max ratio
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

	// Query health: based on latency and sequential scans
	if raw.Queries != nil {
		queryHealth := 1.0
		hasQueryMetrics := false

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

		// Penalise for sequential scans (indicate missing indexes)
		if raw.Queries.SequentialScans != nil {
			seqScans := float64(*raw.Queries.SequentialScans)
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

	// Storage health: based on used/total ratio
	if raw.Storage != nil && raw.Storage.UsedSizeBytes != nil && raw.Storage.TotalSizeBytes != nil {
		used := float64(*raw.Storage.UsedSizeBytes)
		total := float64(*raw.Storage.TotalSizeBytes)

		if total > 0 {
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

	// Cache health: direct hit rate
	if raw.Cache != nil && raw.Cache.HitRate != nil {
		normalised.CacheHealth = *raw.Cache.HitRate

		normalised.Measurements.CacheHitRate = raw.Cache.HitRate
		normalised.Measurements.CacheHitCount = raw.Cache.HitCount
		normalised.Measurements.CacheMissCount = raw.Cache.MissCount

		healthScores = append(healthScores, normalised.CacheHealth)
	} else {
		normalised.CacheHealth = 1.0
	}

	// Overall health: average of available health scores
	if len(healthScores) > 0 {
		var total float64
		for _, score := range healthScores {
			total += score
		}
		normalised.HealthScore = total / float64(len(healthScores))
	} else {
		normalised.HealthScore = 1.0
	}

	// Pass through extended metrics and labels
	if raw.ExtendedMetrics != nil {
		normalised.ExtendedMetrics = raw.ExtendedMetrics
	}

	if raw.Labels != nil {
		normalised.Labels = raw.Labels
	}

	// Calculate deltas from previous collection
	n.calculateDeltas(normalised)
	n.previousMetrics[normalised.DatabaseID] = normalised

	return normalised, nil
}

// calculateDeltas computes metric changes between collection cycles.
func (n *MySQLNormaliser) calculateDeltas(current *NormalisedMetrics) {
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

	// Sequential scans delta
	if current.Measurements.SequentialScans != nil && previous.Measurements.SequentialScans != nil {
		currentVal := float64(*current.Measurements.SequentialScans)
		previousVal := float64(*previous.Measurements.SequentialScans)
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
