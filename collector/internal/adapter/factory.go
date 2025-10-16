package adapter

func NewAdapter(adapterType string, connectionString string) (MetricAdapter, error) {
	switch adapterType {
	case "postgres", "postgresql":
		return NewPostgresAdapter(connectionString), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}
