// Package adapter provides the system with a range of adapters for different databases
// Each adhere to the loose interface.go keeping everything SOLID
package adapter

import (
	"errors"
	"time"
)

// MetricAdapter defines loosely what every DB Adapter will have to do in the future.
type MetricAdapter interface {
	Connect() error
	CollectMetrics() (*RawMetrics, error)
	Close() error
	HealthCheck() error
}

// RawMetrics contains DB Agnostic metrics collected by adapters.
// This struct matches the MetricSnapshot message in gRPC Contract.
type RawMetrics struct {
	// Metadata: identifies the source and collection time
	DatabaseID   string // Unique database identifier
	Timestamp    int64  // Unix timestamp in seconds
	DatabaseType string // Database type (e.g., "postgresql", "mysql")

	// System-level metrics: host machine resources
	CPUPercent          float64 // System CPU usage (0.0-100.0)
	MemoryPercent       float64 // System memory usage (0.0-100.0)
	DiskIOReadMBPerSec  float64 // Disk read throughput (MB/sec)
	DiskIOWriteMBPerSec float64 // Disk write throughput (MB/sec)

	// Connection metrics: database connection pool state
	ActiveConnections    int32   // Currently active connections
	IdleConnections      int32   // Idle connections in pool
	MaxConnections       int32   // Maximum connection limit
	ConnectionWaitTimeMS float64 // Average connection wait time (ms)

	// Query performance: statistical distribution of query times
	QueryLatencyP50MS float64 // Median query time (50% faster)
	QueryLatencyP95MS float64 // 95th percentile query time
	QueryLatencyP99MS float64 // 99th percentile (catches slow outliers)
	QueriesPerSecond  float64 // Query throughput

	// Cache metrics: memory cache efficiency
	CacheHitRate float64 // Cache hit ratio (0.0-1.0)
	CacheSizeMB  float64 // Current cache size (MB)

	// Error tracking: failure rate
	ErrorsPerSecond float64 // Error rate (errors/sec)

	// Extensible fields: database-specific data
	ExtendedMetrics map[string]float64 // DB-specific numeric metrics (e.g., "pg.sequential_scans")
	Labels          map[string]string  // Non-numeric metadata tags (e.g., "region": "us-east-1")

}

var (
	// NotConnected - Connect() not called | failed
	ErrNotConnected = errors.New("adapter: not connected to database")

	// ConnectionLost - self explanatory for now
	ErrConnectionLost = errors.New("adapter: database connection lost")

	// UnsupportedDatabase - If connected DB is unsupported fallback error
	ErrUnsupportedDatabase = errors.New("adapter: unsupported database type")
)

// NewRawMetrics creates RawMetrics with prefilled passed data, maps initialised
func NewRawMetrics(databaseID, databaseType string) *RawMetrics {
	return &RawMetrics{
		DatabaseID:      databaseID,
		DatabaseType:    databaseType,
		Timestamp:       time.Now().Unix(),
		ExtendedMetrics: map[string]float64{},
		Labels:          make(map[string]string),
	}
}
