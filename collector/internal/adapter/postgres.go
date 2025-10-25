package adapter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdapter struct {
	connectionString string
	databaseId       string
	pool             *pgxpool.Pool
}

func NewPostgresAdapter(connectionString string, databaseId string) *PostgresAdapter {
	return &PostgresAdapter{
		connectionString: connectionString,
		databaseId:       databaseId,
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

	metrics := NewRawMetrics(p.databaseId, "postgresql")

	activeConn, err := p.getActiveConnections(ctx)
	if err != nil {
		return nil, err
	}

	idleConn, err := p.getIdleConnections(ctx)
	if err != nil {
		return nil, err
	}

	maxConn, err := p.getMaxConnections(ctx)
	if err != nil {
		return nil, err
	}

	metrics.Connections = &ConnectionMetrics{
		Active: &activeConn,
		Idle:   &idleConn,
		Max:    &maxConn,
		// Waiting nil - Need additional query
	}

	dbSizeBytes, err := p.getDatabaseSizeBytes(ctx)
	if err != nil {
		return nil, err
	}

	metrics.Storage = &StorageMetrics{
		UsedSizeBytes: &dbSizeBytes,
		// Total nil - need filesystem query
		// Free space same
	}

	dbSizeMB := float64(dbSizeBytes) / (1024 * 1024)
	metrics.ExtendedMetrics["pg.database_size_mb"] = dbSizeMB

	cacheHitRate, err := p.getCacheHitRate(ctx)
	if err != nil {
		return nil, err
	}
	metrics.Cache = &CacheMetrics{
		HitRate: &cacheHitRate,
		//Hit/Miss nil
	}

	seqScans, err := p.getSequentialScans(ctx)
	if err != nil {
		return nil, err
	}

	metrics.Queries = &QueryMetrics{
		SequentialScans: &seqScans,
	}

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

func (p *PostgresAdapter) getDatabaseSizeBytes(ctx context.Context) (int64, error) {
	var sizeBytes int64
	query := "SELECT pg_database_size(current_database())"

	err := p.pool.QueryRow(ctx, query).Scan(&sizeBytes)
	if err != nil {
		return 00, fmt.Errorf("failed to get db size: %w", err)
	}

	return sizeBytes, nil
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

func (p *PostgresAdapter) getSequentialScans(ctx context.Context) (int32, error) {
	var seqScans int64
	query := `
        SELECT COALESCE(SUM(seq_scan), 0)
        FROM pg_stat_user_tables
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    `

	err := p.pool.QueryRow(ctx, query).Scan(&seqScans)
	if err != nil {
		return 0, fmt.Errorf("failed to get sequential scans: %w", err)
	}

	return int32(seqScans), nil
}
