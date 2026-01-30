// Package adapter provides the system with a range of adapters for different databases
package adapter

import (
	"errors"
)

// MetricAdapter defines loosely what every DB Adapter will have to do in the future.
type MetricAdapter interface {
	Connect() error
	CollectMetrics() (*RawMetrics, error)
	Close() error
	HealthCheck() error
	GetUnavailableFeatures() []string
}

var (
	// NotConnected - Connect() not called | failed
	ErrNotConnected = errors.New("adapter: not connected to database")

	// ConnectionLost - self explanatory for now
	ErrConnectionLost = errors.New("adapter: database connection lost")

	// UnsupportedDatabase - If connected DB is unsupported fallback error
	ErrUnsupportedDatabase = errors.New("adapter: unsupported database type")
)
