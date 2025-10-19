package adapter

import "time"

type RawMetrics struct {
	// Metadata: identifies the source and collection time
	DatabaseID   string
	DatabaseType string
	Timestamp    int64

	// Optional Metrics: Not every database will have each
	Connections *ConnectionMetrics
	Queries     *QueryMetrics
	Storage     *StorageMetrics
	Cache       *CacheMetrics

	// Extensible fields: database-specific data
	ExtendedMetrics map[string]float64
	Labels          map[string]string

	// For databases that expose structured logs
	Logs []LogEntry
}

// Track connection pools if DB has them
type ConnectionMetrics struct {
	Active  *int32
	Idle    *int32
	Max     *int32
	Waiting *int32
}

// QueryMetrics tracks query performance
type QueryMetrics struct {
	QueriesPerSecond *float64
	AvgLatencyMs     *float64
	P50LatencyMs     *float64
	P95LatencyMs     *float64
	P99LatencyMs     *float64
	SlowQueries      []SlowQuery
	SequentialScans  *int32 // PG Concept
}

// Storage tracks disk usage
type StorageMetrics struct {
	TotalSizeBytes *int64
	UsedSizeBytes  *int64
	FreeSpaceBytes *int64
	IndexSizeBytes *int64
	TableSizeBytes *int64
}

// Cache tracks caching
type CacheMetrics struct {
	HitRate        *float64
	HitCount       *int64
	MissCount      *int64
	CacheSizeBytes *int64
}

// SlowQuery is a query that exceeds performance threshold
type SlowQuery struct {
	Query      string
	DurationMs float64
	Timestamp  int64
	Source     string
}

// Log represents a DB log entry
type LogEntry struct {
	Timestamp int64
	Level     string
	Message   string
	Metadata  map[string]string
}

func NewRawMetrics(databaseID, databaseType string) *RawMetrics {
	return &RawMetrics{
		DatabaseID:      databaseID,
		DatabaseType:    databaseType,
		Timestamp:       time.Now().Unix(),
		ExtendedMetrics: make(map[string]float64),
		Labels:          make(map[string]string),
	}
}
