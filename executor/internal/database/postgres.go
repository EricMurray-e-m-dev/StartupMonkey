package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdapter struct {
	pool         *pgxpool.Pool
	databaseName string
}

func NewPostgresAdapter(ctx context.Context, connectionString, databaseName string) (*PostgresAdapter, error) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresAdapter{
		pool:         pool,
		databaseName: databaseName,
	}, nil
}

func (p *PostgresAdapter) CreateIndex(ctx context.Context, params IndexParams) error {
	exists, err := p.IndexExists(ctx, params.IndexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return ErrIndexAlreadyExists
	}

	columns := strings.Join(params.ColumnNames, ", ")

	var query string
	indexType := "INDEX"
	if params.Unique {
		indexType = "UNIQUE INDEX"
	}

	if params.Concurrent {
		query = fmt.Sprintf("CREATE %s CONCURRENTLY IF NOT EXISTS %s ON %s (%s)", indexType, params.IndexName, params.TableName, columns)
	} else {
		query = fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s)", indexType, params.IndexName, params.TableName, columns)
	}

	_, err = p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func (p *PostgresAdapter) DropIndex(ctx context.Context, indexName string) error {
	exists, err := p.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if !exists {
		return nil
	}

	query := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", indexName)

	_, err = p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

func (p *PostgresAdapter) IndexExists(ctx context.Context, indexName string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)"

	var exists bool
	err := p.pool.QueryRow(ctx, query, indexName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}

	return exists, nil
}

func (p *PostgresAdapter) GetCapabilities() Capabilities {
	return Capabilities{
		SupportsIndexes:           true,
		SupportsConcurrentIndexes: true, // Postgres 8.2+
		SupportsUniqueIndex:       true,
		SupportsMultiColumnIndex:  true,
	}
}

func (p *PostgresAdapter) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}
