// Package adapter provides database-specific metric collection implementations.
package adapter

import "time"

// RawMetrics contains unprocessed metrics collected from a database.
// These are normalised before being sent to the Analyser service.
type RawMetrics struct {
	// Metadata
	DatabaseID   string
	DatabaseType string
	Timestamp    int64

	// Core metric categories (nil if not available for this database type)
	Connections *ConnectionMetrics
	Queries     *QueryMetrics
	Storage     *StorageMetrics
	Cache       *CacheMetrics

	// Extensible fields for database-specific data
	ExtendedMetrics map[string]float64
	Labels          map[string]string

	// Structured logs (future use)
	Logs []LogEntry
}

// ConnectionMetrics tracks database connection pool statistics.
type ConnectionMetrics struct {
	Active  *int32
	Idle    *int32
	Max     *int32
	Waiting *int32
}

// QueryMetrics tracks query performance statistics.
type QueryMetrics struct {
	QueriesPerSecond *float64
	AvgLatencyMs     *float64
	P50LatencyMs     *float64
	P95LatencyMs     *float64
	P99LatencyMs     *float64
	SlowQueries      []SlowQuery
	SequentialScans  *int32
}

// StorageMetrics tracks database disk usage.
type StorageMetrics struct {
	TotalSizeBytes *int64
	UsedSizeBytes  *int64
	FreeSpaceBytes *int64
	IndexSizeBytes *int64
	TableSizeBytes *int64
}

// CacheMetrics tracks database caching performance.
type CacheMetrics struct {
	HitRate        *float64
	HitCount       *int64
	MissCount      *int64
	CacheSizeBytes *int64
}

// SlowQuery represents a query that exceeded performance thresholds.
type SlowQuery struct {
	Query      string
	DurationMs float64
	Timestamp  int64
	Source     string
}

// LogEntry represents a structured database log entry.
type LogEntry struct {
	Timestamp int64
	Level     string
	Message   string
	Metadata  map[string]string
}

// NewRawMetrics creates a new RawMetrics instance with initialised maps.
func NewRawMetrics(databaseID, databaseType string) *RawMetrics {
	return &RawMetrics{
		DatabaseID:      databaseID,
		DatabaseType:    databaseType,
		Timestamp:       time.Now().Unix(),
		ExtendedMetrics: make(map[string]float64),
		Labels:          make(map[string]string),
	}
}
