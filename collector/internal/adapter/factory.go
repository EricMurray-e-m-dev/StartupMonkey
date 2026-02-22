// Package adapter provides database-specific metric collection implementations.
package adapter

// NewAdapter creates a MetricAdapter for the specified database type.
func NewAdapter(adapterType string, connectionString string, databaseID string) (MetricAdapter, error) {
	switch adapterType {
	case "postgres", "postgresql":
		return NewPostgresAdapter(connectionString, databaseID), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}
