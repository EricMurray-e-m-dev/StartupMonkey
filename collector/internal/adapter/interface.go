// Package adapter provides database-specific metric collection implementations.
package adapter

import (
	"errors"
)

// MetricAdapter defines the interface that all database adapters must implement.
type MetricAdapter interface {
	Connect() error
	CollectMetrics() (*RawMetrics, error)
	Close() error
	HealthCheck() error
	GetUnavailableFeatures() []string
}

var (
	// ErrNotConnected is returned when Connect() has not been called or failed.
	ErrNotConnected = errors.New("adapter: not connected to database")

	// ErrConnectionLost is returned when the database connection is lost.
	ErrConnectionLost = errors.New("adapter: database connection lost")

	// ErrUnsupportedDatabase is returned when an unknown database type is requested.
	ErrUnsupportedDatabase = errors.New("adapter: unsupported database type")
)
