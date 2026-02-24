// Package normaliser converts raw database metrics into normalised health scores.
package normaliser

// NormalisedMetrics contains processed metrics with health scores.
// This structure aligns with the MetricSnapshot proto message.
type NormalisedMetrics struct {
	// Metadata
	DatabaseID   string `json:"database_id"`
	DatabaseType string `json:"database_type"`
	Timestamp    int64  `json:"timestamp"`

	// Normalised health scores (0.0 - 1.0)
	HealthScore      float64 `json:"health_score"`
	ConnectionHealth float64 `json:"connection_health"`
	QueryHealth      float64 `json:"query_health"`
	StorageHealth    float64 `json:"storage_health"`
	CacheHealth      float64 `json:"cache_health"`

	// Available metrics for this collection cycle
	AvailableMetrics []string `json:"available_metrics"`

	// Raw measurements passed through to Analyser
	Measurements Measurements `json:"measurements"`

	// Delta calculations between collection cycles
	MetricDeltas     map[string]float64 `json:"metric_deltas"`
	TimeDeltaSeconds float64            `json:"time_delta_seconds"`

	// Database-specific extended data
	ExtendedMetrics map[string]float64 `json:"extended_metrics"`
	Labels          map[string]string  `json:"labels"`
}

// Measurements contains raw metric values.
// This structure aligns with the Measurements proto message.
type Measurements struct {
	// Connection metrics
	ActiveConnections  *int32 `json:"active_connections,omitempty"`
	IdleConnections    *int32 `json:"idle_connections,omitempty"`
	MaxConnections     *int32 `json:"max_connections,omitempty"`
	WaitingConnections *int32 `json:"waiting_connections,omitempty"`

	// Query performance metrics
	AvgQueryLatencyMs *float64 `json:"avg_query_latency_ms,omitempty"`
	P50QueryLatencyMs *float64 `json:"p50_query_latency_ms,omitempty"`
	P95QueryLatencyMs *float64 `json:"p95_query_latency_ms,omitempty"`
	P99QueryLatencyMs *float64 `json:"p99_query_latency_ms,omitempty"`
	SlowQueryCount    *int32   `json:"slow_query_count,omitempty"`
	SequentialScans   *int32   `json:"sequential_scans,omitempty"`

	// Storage metrics
	UsedStorageBytes  *int64 `json:"used_storage_bytes,omitempty"`
	TotalStorageBytes *int64 `json:"total_storage_bytes,omitempty"`
	FreeStorageBytes  *int64 `json:"free_storage_bytes,omitempty"`

	// Cache metrics
	CacheHitRate   *float64 `json:"cache_hit_rate,omitempty"`
	CacheHitCount  *int64   `json:"cache_hit_count,omitempty"`
	CacheMissCount *int64   `json:"cache_miss_count,omitempty"`
}
