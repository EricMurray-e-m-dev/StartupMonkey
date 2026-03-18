package normaliser

import (
	"math"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
)

// MongoDBNormaliser converts raw MongoDB metrics to normalised health scores.
type MongoDBNormaliser struct {
	previousMetrics map[string]*NormalisedMetrics
}

// NewMongoDBNormaliser creates a new MongoDB normaliser.
func NewMongoDBNormaliser() *MongoDBNormaliser {
	return &MongoDBNormaliser{
		previousMetrics: make(map[string]*NormalisedMetrics),
	}
}

// Normalise converts raw MongoDB metrics to normalised health scores.
// Health scores range from 0.0 (critical) to 1.0 (healthy).
func (n *MongoDBNormaliser) Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error) {
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

		healthScores = append(healthScores, normalised.ConnectionHealth)
	} else {
		normalised.ConnectionHealth = 1.0
	}

	// Query health: based on collection scans
	if raw.Queries != nil {
		queryHealth := 1.0

		if raw.Queries.SequentialScans != nil {
			normalised.Measurements.SequentialScans = raw.Queries.SequentialScans

			seqScans := float64(*raw.Queries.SequentialScans)
			// Reduce health by 10% per 100 collection scans (max 50% reduction)
			seqScanPenalty := math.Min(0.5, (seqScans/100.0)*0.1)
			queryHealth = math.Max(0, queryHealth-seqScanPenalty)

			healthScores = append(healthScores, queryHealth)
		}

		normalised.QueryHealth = queryHealth
	} else {
		normalised.QueryHealth = 1.0
	}

	// Storage health: MongoDB doesn't report total size the same way
	normalised.StorageHealth = 1.0

	// Cache health: WiredTiger cache hit rate
	if raw.Cache != nil && raw.Cache.HitRate != nil {
		normalised.CacheHealth = *raw.Cache.HitRate
		normalised.Measurements.CacheHitRate = raw.Cache.HitRate
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
func (n *MongoDBNormaliser) calculateDeltas(current *NormalisedMetrics) {
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

	// Sequential scans (collection scans) delta
	if current.Measurements.SequentialScans != nil && previous.Measurements.SequentialScans != nil {
		currentVal := float64(*current.Measurements.SequentialScans)
		previousVal := float64(*previous.Measurements.SequentialScans)
		delta := currentVal - previousVal

		if delta < 0 {
			delta = 0
		}

		current.MetricDeltas["sequential_scans"] = delta
	}
}
