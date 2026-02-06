package database

import (
	"context"
	"fmt"
	"regexp"
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

func (p *PostgresAdapter) GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error) {
	config := make(map[string]string)

	for _, param := range parameters {
		var value string
		query := fmt.Sprintf("SHOW %s", param)
		err := p.pool.QueryRow(ctx, query).Scan(&value)
		if err != nil {
			return nil, fmt.Errorf("failed to get config for: %s: %w", param, err)
		}
		config[param] = value
	}

	return config, nil
}

func (p *PostgresAdapter) SetConfig(ctx context.Context, changes map[string]string) error {

	for param, value := range changes {
		query := fmt.Sprintf("ALTER SYSTEM SET %s = '%s'", param, value)
		_, err := p.pool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to set %s to %s: %w", param, value, err)
		}
	}

	_, err := p.pool.Exec(ctx, "SELECT pg_reload_conf()")
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	return nil
}

func (p *PostgresAdapter) GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]SlowQuery, error) {
	query := `
		SELECT 
			query,
			mean_exec_time,
			calls
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := p.pool.Query(ctx, query, thresholdMs, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query slow queries: %w", err)
	}
	defer rows.Close()

	var slowQueries []SlowQuery

	for rows.Next() {
		var rawQuery string
		var execTime float64
		var calls int32

		if err := rows.Scan(&rawQuery, &execTime, &calls); err != nil {
			return nil, fmt.Errorf("failed to scan slow queries: %w", err)
		}

		sanitised := sanitiseQuery(rawQuery)

		issueType, recommendation := analyseQuery(rawQuery)

		slowQueries = append(slowQueries, SlowQuery{
			QueryPattern:    sanitised,
			ExecutionTimeMs: execTime,
			CallCount:       calls,
			IssueType:       issueType,
			Recommendation:  recommendation,
		})
	}
	return slowQueries, nil
}

func (p *PostgresAdapter) GetCapabilities() Capabilities {
	return Capabilities{
		SupportsIndexes:              true,
		SupportsConcurrentIndexes:    true,
		SupportsUniqueIndex:          true,
		SupportsMultiColumnIndex:     true,
		SupportsConfigTuning:         true,
		SupportsRuntimeConfigChanges: true,
		SupportsVacuum:               true,
		SupportsQueryTermination:     true,
	}
}

func (p *PostgresAdapter) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

// Helper: Sanitise queries
func sanitiseQuery(query string) string {
	// Replace string literals
	query = regexp.MustCompile(`'[^']*'`).ReplaceAllString(query, "?")

	// Replace numbers
	query = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(query, "?")

	// Truncate if too long
	if len(query) > 200 {
		query = query[:200] + "..."
	}

	// Remove extra whitespace
	query = strings.Join(strings.Fields(query), " ")

	return query
}

// Helper: Analyse query to determine issue type
func analyseQuery(query string) (issueType string, recommendation string) {
	queryLower := strings.ToLower(query)

	// Check for sequential scans (common with missing indexes)
	if strings.Contains(queryLower, "seq scan") {
		return "sequential_scan", "Consider adding an index on the filtered columns"
	}

	// Check for ORDER BY without index
	if strings.Contains(queryLower, "order by") && !strings.Contains(queryLower, "index") {
		return "missing_index", "Add index on ORDER BY columns for faster sorting"
	}

	// Check for complex joins
	if strings.Count(queryLower, "join") >= 3 {
		return "complex_join", "Consider simplifying joins or adding indexes on join columns"
	}

	// Check for SELECT *
	if strings.Contains(queryLower, "select *") {
		return "inefficient_select", "Select only required columns instead of SELECT *"
	}

	// Default
	return "high_latency", "Review query execution plan with EXPLAIN ANALYZE"
}

func (p *PostgresAdapter) VacuumTable(ctx context.Context, tableName string) error {
	// VACUUM cannot run inside a transaction, so we use a simple connection
	query := fmt.Sprintf("VACUUM ANALYZE %s", tableName)

	_, err := p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to vacuum table %s: %w", tableName, err)
	}

	return nil
}

func (p *PostgresAdapter) GetDeadTuples(ctx context.Context, tableName string) (int64, error) {
	query := `
		SELECT n_dead_tup 
		FROM pg_stat_user_tables 
		WHERE relname = $1
	`

	var deadTuples int64
	err := p.pool.QueryRow(ctx, query, tableName).Scan(&deadTuples)
	if err != nil {
		return 0, fmt.Errorf("failed to get dead tuples for %s: %w", tableName, err)
	}

	return deadTuples, nil
}

func (p *PostgresAdapter) TerminateQuery(ctx context.Context, pid int32, graceful bool) error {
	var success bool
	var query string

	if graceful {
		// pg_cancel_backend sends SIGINT - query can handle cancellation gracefully
		query = "SELECT pg_cancel_backend($1)"
	} else {
		// pg_terminate_backend sends SIGTERM - forceful termination
		query = "SELECT pg_terminate_backend($1)"
	}

	err := p.pool.QueryRow(ctx, query, pid).Scan(&success)
	if err != nil {
		return fmt.Errorf("failed to terminate query: %w", err)
	}

	if !success {
		return fmt.Errorf("failed to terminate PID %d: process not found or permission denied", pid)
	}

	return nil
}
