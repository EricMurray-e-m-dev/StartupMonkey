package normaliser

type NormalisedMetrics struct {
	DatabaseID   string `json:"database_id"`
	DatabaseType string `json:"database_type"`
	Timestamp    int64  `json:"timestamp"`

	// Normalised health scores (0.0 - 1.0)
	HealthScore      float64 `json:"health_score"`
	ConnectionHealth float64 `json:"connection_health"`
	QueryHealth      float64 `json:"query_health"`
	StorageHealth    float64 `json:"storage_health"`
	CacheHealth      float64 `json:"cache_health"`

	// What metrics were available
	AvailableMetrics []string `json:"available_metrics"`

	// Raw measurements
	Measurements Measurements `json:"measurements"`

	// Deltas
	MetricDeltas     map[string]float64 `json:"metric_deltas"`
	TimeDeltaSeconds float64            `json:"time_delta_seconds"`

	ExtendedMetrics map[string]float64 `json:"extended_metrics"`
	Labels          map[string]string  `json:"labels"`
}

type Measurements struct {
	// Connections (nil if not available)
	ActiveConnections  *int32 `json:"active_connections,omitempty"`
	IdleConnections    *int32 `json:"idle_connections,omitempty"`
	MaxConnections     *int32 `json:"max_connections,omitempty"`
	WaitingConnections *int32 `json:"waiting_connections,omitempty"`

	// Queries (nil if not available)
	AvgQueryLatencyMs *float64 `json:"avg_query_latency_ms,omitempty"`
	P50QueryLatencyMs *float64 `json:"p50_query_latency_ms,omitempty"`
	P95QueryLatencyMs *float64 `json:"p95_query_latency_ms,omitempty"`
	P99QueryLatencyMs *float64 `json:"p99_query_latency_ms,omitempty"`
	SlowQueryCount    *int32   `json:"slow_query_count,omitempty"`
	SequentialScans   *int32   `json:"sequential_scans,omitempty"`

	// Storage (nil if not available)
	UsedStorageBytes  *int64 `json:"used_storage_bytes,omitempty"`
	TotalStorageBytes *int64 `json:"total_storage_bytes,omitempty"`
	FreeStorageBytes  *int64 `json:"free_storage_bytes,omitempty"`

	// Cache (nil if not available)
	CacheHitRate   *float64 `json:"cache_hit_rate,omitempty"`
	CacheHitCount  *int64   `json:"cache_hit_count,omitempty"`
	CacheMissCount *int64   `json:"cache_miss_count,omitempty"`
}
