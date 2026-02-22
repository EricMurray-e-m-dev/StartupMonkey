// Package normaliser converts raw database metrics into normalised health scores.
package normaliser

import "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"

// Normaliser defines the interface for converting raw metrics to normalised form.
type Normaliser interface {
	Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error)
}

// NewNormaliser creates a Normaliser for the specified database type.
// Returns nil for unsupported database types.
func NewNormaliser(databaseType string) Normaliser {
	switch databaseType {
	case "postgres", "postgresql":
		return NewPostgresNormaliser()
	default:
		return nil
	}
}
