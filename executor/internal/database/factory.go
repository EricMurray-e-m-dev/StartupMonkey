package database

import (
	"context"
	"fmt"
	"strings"
)

// NewAdapter creates the appropriate database adapter based on database type.
func NewAdapter(ctx context.Context, databaseType, connectionString, databaseID string) (DatabaseAdapter, error) {
	switch strings.ToLower(databaseType) {
	case "postgres", "postgresql":
		return NewPostgresAdapter(ctx, connectionString, databaseID)
	case "mysql", "mariadb":
		return NewMySQLAdapter(ctx, connectionString, databaseID)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}
}
