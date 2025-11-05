package normaliser

type NormalisedMetrics struct {
	DatabaseID   string
	DatabaseType string
	Timestamp    int64

	// Normalised health scores (0.0 - 1.0)
	HealthScore      float64
	ConnectionHealth float64
	QueryHealth      float64
	StorageHealth    float64
	CacheHealth      float64

	// What metrics were available
	AvailableMetrics []string

	// Raw measurements
	Measurements Measurements

	ExtendedMetrics map[string]float64
	Labels          map[string]string
}

type Measurements struct {
	// Connections (nil if not available)
	ActiveConnections  *int32
	IdleConnections    *int32
	MaxConnections     *int32
	WaitingConnections *int32

	// Queries (nil if not available)
	AvgQueryLatencyMs *float64
	P50QueryLatencyMs *float64
	P95QueryLatencyMs *float64
	P99QueryLatencyMs *float64
	SlowQueryCount    *int32
	SequentialScans   *int32

	// Storage (nil if not available)
	UsedStorageBytes  *int64
	TotalStorageBytes *int64
	FreeStorageBytes  *int64

	// Cache (nil if not available)
	CacheHitRate   *float64
	CacheHitCount  *int64
	CacheMissCount *int64
}
