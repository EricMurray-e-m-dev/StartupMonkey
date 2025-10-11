package adapter

import (
	"context"
	"fmt"
	"strconv"

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

func (p *PostgresAdapter) CollectMetrics() (*RawMetrics, error) {
	if p.pool == nil {
		return nil, ErrNotConnected
	}

	ctx := context.Background()

	metrics := NewRawMetrics("postgres-1", "postgresql")

	activeConn, err := p.getActiveConnections(ctx)
	if err != nil {
		return nil, err
	}
	metrics.ActiveConnections = activeConn

	idleConn, err := p.getIdleConnections(ctx)
	if err != nil {
		return nil, err
	}
	metrics.IdleConnections = idleConn

	maxConn, err := p.getMaxConnections(ctx)
	if err != nil {
		return nil, err
	}
	metrics.MaxConnections = maxConn

	dbSize, err := p.getDatabaseSizeMB(ctx)
	if err != nil {
		return nil, err
	}
	metrics.ExtendedMetrics["pg.database_size_mb"] = dbSize

	cacheHitRate, err := p.getCacheHitRate(ctx)
	if err != nil {
		return nil, err
	}
	metrics.CacheHitRate = cacheHitRate

	// TODO: System metrics (CPU, memory, disk I/O) should be collected by
	// a separate SystemMetricsCollector in the Collector orchestrator.

	// TODO: Query performance metrics (p50/p95/p99 latency) require
	// pg_stat_statements extension.

	return metrics, nil

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
	var countString string
	query := "SHOW max_connections"

	err := p.pool.QueryRow(ctx, query).Scan(&countString)
	if err != nil {
		return 0, fmt.Errorf("failed to get max connections: %w", err)
	}

	count, err := strconv.ParseInt(countString, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse max_connections: %w", err)
	}

	return int32(count), nil
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
