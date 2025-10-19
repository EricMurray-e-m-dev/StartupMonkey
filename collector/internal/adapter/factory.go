package adapter

func NewAdapter(adapterType string, connectionString string, databaseId string) (MetricAdapter, error) {
	switch adapterType {
	case "postgres", "postgresql":
		return NewPostgresAdapter(connectionString, databaseId), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}
