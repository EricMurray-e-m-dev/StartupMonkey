package adapter

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdapter struct {
	connectionString string
	pool             *pgxpool.Pool
}

func NewPostgresAdapter(connectionString string) *PostgresAdapter {
	return &PostgresAdapter{
		connectionString: connectionString,
		pool:             nil,
	}
}

func (p *PostgresAdapter) Connect() error {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, p.connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	p.pool = pool
	return nil
}

func (p *PostgresAdapter) CollectMetrics() (_ *RawMetrics, _ error) {
	panic("not implemented") // TODO: Implement
}

func (p *PostgresAdapter) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

func (p *PostgresAdapter) HealthCheck() error {
	if p.pool == nil {
		return ErrNotConnected
	}

	ctx := context.Background()
	err := p.pool.Ping(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

func (p *PostgresAdapter) getActiveConnections(ctx context.Context) (int32, error) {
	var count int32
	query := "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"

	err := p.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active connections: %w", err)
	}

	return count, nil
}

func (p *PostgresAdapter) getIdleConnections(ctx context.Context) (int32, error) {
	var count int32
	query := "SELECT count(*) FROM pg_stat_activity WHERE state = 'idle'"

	err := p.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get idle connections: %w", err)
	}

	return count, nil
}

func (p *PostgresAdapter) getMaxConnections(ctx context.Context) (int32, error) {
	var count int32
	query := "SHOW max_connections"

	err := p.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get max connections: %w", err)
	}

	return count, nil
}

func (p *PostgresAdapter) getDatabaseSizeMB(ctx context.Context) (float64, error) {
	var size int64
	query := "SELECT pg_database_size(current_database())"

	err := p.pool.QueryRow(ctx, query).Scan(&size)
	if err != nil {
		return 00, fmt.Errorf("failed to get db size: %w", err)
	}

	sizeMB := float64(size) / (1024 * 1024)
	return sizeMB, nil
}

func (p *PostgresAdapter) getCacheHitRate(ctx context.Context) (float64, error) {
	var blkReads, blksHit int64

	query := `
		SELECT
			sum(blks_read) as blks_read,
			sum(blks_hit) as blks_hit
		FROM pg_stat_database
		WHERE datname = current_database()
	`

	err := p.pool.QueryRow(ctx, query).Scan(&blkReads, &blksHit)
	if err != nil {
		return 0, fmt.Errorf("failed to get cache stats: %w", err)
	}

	total := blkReads + blksHit
	if total == 0 {
		return 0, nil
	}

	hitRate := float64(blkReads) / float64(blksHit)
	return hitRate, nil
}
