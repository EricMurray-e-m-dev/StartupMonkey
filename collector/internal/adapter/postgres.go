package adapter

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdapter struct {
	connectionString          string
	databaseId                string
	pool                      *pgxpool.Pool
	pgStatStatementsAvailable bool
}

type TabelScanStat struct {
	TableName  string
	SeqScans   int64
	SeqTupRead int64
	IdxScans   int64
}

type TableBloatStat struct {
	TableName      string
	LiveTuples     int64
	DeadTuples     int64
	BloatRatio     float64
	LastVacuum     *time.Time
	LastAutoVacuum *time.Time
}

type LongRunningQuery struct {
	PID          int32
	Username     string
	DatabaseName string
	Query        string
	State        string
	DurationSecs float64
	WaitEvent    *string
}

type IdleTransaction struct {
	PID              int32
	Username         string
	DatabaseName     string
	Query            string
	IdleDurationSecs float64
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

	// Ensure pg_stat_statements is available
	if err := p.ensurePgStatStatements(ctx); err != nil {
		log.Printf("Warning: pg_stat_statements setup issue: %v", err)
		// Don't fail connection - graceful degradation
	}

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

	tableStats, err := p.getTableScans(ctx)
	if err != nil {
		fmt.Printf("failed to get table stats: %v\n", err)
	} else if len(tableStats) > 0 {
		worstTable := tableStats[0]

		for _, table := range tableStats {
			prefix := fmt.Sprintf("pg.table.%s", table.TableName)
			metrics.ExtendedMetrics[prefix+".seq_scans"] = float64(table.SeqScans)
			metrics.ExtendedMetrics[prefix+".seq_tup_read"] = float64(table.SeqTupRead)
			metrics.ExtendedMetrics[prefix+".idx_scans"] = float64(table.IdxScans)
		}

		metrics.Labels["pg.worst_seq_scan_table"] = worstTable.TableName

		recommendedColumns, err := p.analyseSlowQueries(ctx, worstTable.TableName)
		if err != nil {
			fmt.Printf("warning: could not analyse queries: %v\n", err)
		} else if len(recommendedColumns) > 0 {
			metrics.Labels["pg.recommended_index_column"] = recommendedColumns[0]
		}
	}

	// Table bloat stats
	bloatStats, err := p.getTableBloat(ctx)
	if err != nil {
		fmt.Printf("failed to get table bloat stats: %v\n", err)
	} else if len(bloatStats) > 0 {
		for _, table := range bloatStats {
			prefix := fmt.Sprintf("pg.table.%s", table.TableName)
			metrics.ExtendedMetrics[prefix+".live_tuples"] = float64(table.LiveTuples)
			metrics.ExtendedMetrics[prefix+".dead_tuples"] = float64(table.DeadTuples)
			metrics.ExtendedMetrics[prefix+".bloat_ratio"] = table.BloatRatio
		}

		// Track worst bloated table
		// Track worst bloated table
		worstBloat := bloatStats[0]
		if worstBloat.DeadTuples > 0 {
			metrics.Labels["pg.worst_bloat_table"] = worstBloat.TableName
			metrics.ExtendedMetrics["pg.worst_bloat_ratio"] = worstBloat.BloatRatio
		}
	}

	// Long-running queries
	longQueries, err := p.getLongRunningQueries(ctx, 10.0) // 10 second threshold for collection
	if err != nil {
		fmt.Printf("failed to get long-running queries: %v\n", err)
	} else {
		metrics.ExtendedMetrics["pg.long_running_query_count"] = float64(len(longQueries))

		if len(longQueries) > 0 {
			worst := longQueries[0]
			metrics.Labels["pg.longest_query_pid"] = fmt.Sprintf("%d", worst.PID)
			metrics.Labels["pg.longest_query_user"] = worst.Username
			metrics.Labels["pg.longest_query_text"] = worst.Query
			metrics.ExtendedMetrics["pg.longest_query_duration_secs"] = worst.DurationSecs
		}
	}

	// Idle transactions
	idleTransactions, err := p.getIdleTransactions(ctx, 60.0) // 1 minute threshold for collection
	if err != nil {
		fmt.Printf("failed to get idle transactions: %v\n", err)
	} else {
		metrics.ExtendedMetrics["pg.idle_transaction_count"] = float64(len(idleTransactions))

		if len(idleTransactions) > 0 {
			worst := idleTransactions[0]
			metrics.Labels["pg.idle_txn_pid"] = fmt.Sprintf("%d", worst.PID)
			metrics.Labels["pg.idle_txn_user"] = worst.Username
			metrics.Labels["pg.idle_txn_query"] = worst.Query
			metrics.ExtendedMetrics["pg.idle_txn_duration_secs"] = worst.IdleDurationSecs
		}
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

func (p *PostgresAdapter) GetUnavailableFeatures() []string {
	var features []string
	if !p.pgStatStatementsAvailable {
		features = append(features, "pg_stat_statements")
	}
	return features
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

	hitRate := float64(blksHit) / float64(total)
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

func (p *PostgresAdapter) getTableScans(ctx context.Context) ([]TabelScanStat, error) {
	query := `
        SELECT 
            relname,
            seq_scan,
            seq_tup_read,
            idx_scan
        FROM pg_stat_user_tables
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
        AND seq_scan > 0
        ORDER BY seq_scan DESC
        LIMIT 10
    `

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TabelScanStat
	for rows.Next() {
		var s TabelScanStat
		if err := rows.Scan(&s.TableName, &s.SeqScans, &s.SeqTupRead, &s.IdxScans); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

func (p *PostgresAdapter) analyseSlowQueries(ctx context.Context, tableName string) ([]string, error) {
	if !p.pgStatStatementsAvailable {
		return nil, fmt.Errorf("pg_stat_statements not available")
	}
	//log.Printf("Analysing Queries from table %s", tableName)
	query := `
		SELECT 
			query,
			calls,
			mean_exec_time,
			total_exec_time
		FROM pg_stat_statements
		WHERE query ILIKE $1
		AND calls > 1
		ORDER BY mean_exec_time DESC
		LIMIT 10
	`

	pattern := fmt.Sprintf("%%FROM %s%%", tableName)
	//log.Printf("search pattern: %s", pattern)

	rows, err := p.pool.Query(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	columnFrequency := make(map[string]int)
	rowCount := 0

	for rows.Next() {
		rowCount++
		var sqlQuery string
		var calls int64
		var meanExecTime, totalExecTime float64

		if err := rows.Scan(&sqlQuery, &calls, &meanExecTime, &totalExecTime); err != nil {
			continue
		}

		log.Printf("found query (calls = %d): %s\n", calls, sqlQuery)

		columns := extractFilteredColumns(sqlQuery)
		//log.Printf("Extracted columns: %v\n", columns)

		for _, col := range columns {
			columnFrequency[col] += int(calls)
		}
	}

	//log.Printf("processed %d queries", rowCount)
	//log.Printf("column freq: %v'\n", columnFrequency)

	var recommendedColumns []string
	for col := range columnFrequency {
		recommendedColumns = append(recommendedColumns, col)
	}

	//log.Printf("recommended cols: %v\n", recommendedColumns)

	return recommendedColumns, nil
}

func extractFilteredColumns(query string) []string {
	var columns []string

	patterns := []string{
		`WHERE\s+(\w+)\s*=`,
		`WHERE\s+(\w+)\s+IN`,
		`WHERE\s+(\w+)\s*[><]`,
		`AND\s+(\w+)\s*=`,
		`AND\s+(\w+)\s+IN`,
		`AND\s+(\w+)\s*[><]`,
	}

	queryUpper := strings.ToUpper(query)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(queryUpper, -1)
		for _, match := range matches {
			if len(match) > 1 {
				columns = append(columns, strings.ToLower(match[1]))
			}
		}
	}

	return columns
}

func (p *PostgresAdapter) ensurePgStatStatements(ctx context.Context) error {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements'
		)
	`).Scan(&exists)

	if err != nil {
		p.pgStatStatementsAvailable = false
		return fmt.Errorf("failed to check pg_stat_statements: %w", err)
	}

	if exists {
		p.pgStatStatementsAvailable = true
		return nil
	}

	// Check if preloaded before attempting creation
	var sharedLibs string
	_ = p.pool.QueryRow(ctx, `SHOW shared_preload_libraries`).Scan(&sharedLibs)

	if !strings.Contains(sharedLibs, "pg_stat_statements") {
		p.pgStatStatementsAvailable = false
		return fmt.Errorf("pg_stat_statements not in shared_preload_libraries")
	}

	_, err = p.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_stat_statements`)
	if err != nil {
		p.pgStatStatementsAvailable = false
		return fmt.Errorf("failed to create extension: %w", err)
	}

	p.pgStatStatementsAvailable = true
	log.Printf("pg_stat_statements extension enabled")
	return nil
}

func (p *PostgresAdapter) getTableBloat(ctx context.Context) ([]TableBloatStat, error) {
	query := `
		SELECT 
			relname,
			n_live_tup,
			n_dead_tup,
			last_vacuum,
			last_autovacuum
		FROM pg_stat_user_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		AND n_live_tup > 0
		ORDER BY n_dead_tup DESC
		LIMIT 10
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table bloat: %w", err)
	}
	defer rows.Close()

	var stats []TableBloatStat
	for rows.Next() {
		var s TableBloatStat
		if err := rows.Scan(&s.TableName, &s.LiveTuples, &s.DeadTuples, &s.LastVacuum, &s.LastAutoVacuum); err != nil {
			return nil, err
		}
		if s.LiveTuples > 0 {
			s.BloatRatio = float64(s.DeadTuples) / float64(s.LiveTuples)
		}
		stats = append(stats, s)
	}

	return stats, nil
}

func (p *PostgresAdapter) getLongRunningQueries(ctx context.Context, thresholdSecs float64) ([]LongRunningQuery, error) {
	query := `
		SELECT 
			pid,
			usename,
			datname,
			LEFT(query, 200) as query,
			state,
			EXTRACT(EPOCH FROM (now() - query_start)) as duration_secs,
			wait_event_type
		FROM pg_stat_activity
		WHERE state = 'active'
		AND query NOT LIKE 'autovacuum:%'
		AND pid != pg_backend_pid()
		AND query_start IS NOT NULL
		AND EXTRACT(EPOCH FROM (now() - query_start)) > $1
		ORDER BY duration_secs DESC
		LIMIT 10
	`

	rows, err := p.pool.Query(ctx, query, thresholdSecs)
	if err != nil {
		return nil, fmt.Errorf("failed to query long-running queries: %w", err)
	}
	defer rows.Close()

	var queries []LongRunningQuery
	for rows.Next() {
		var q LongRunningQuery
		if err := rows.Scan(&q.PID, &q.Username, &q.DatabaseName, &q.Query, &q.State, &q.DurationSecs, &q.WaitEvent); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}

	return queries, nil
}

func (p *PostgresAdapter) getIdleTransactions(ctx context.Context, thresholdSecs float64) ([]IdleTransaction, error) {
	query := `
		SELECT 
			pid,
			usename,
			datname,
			LEFT(COALESCE(query, ''), 200) as query,
			EXTRACT(EPOCH FROM (now() - state_change)) as idle_duration_secs
		FROM pg_stat_activity
		WHERE state = 'idle in transaction'
		AND pid != pg_backend_pid()
		AND state_change IS NOT NULL
		AND EXTRACT(EPOCH FROM (now() - state_change)) > $1
		ORDER BY idle_duration_secs DESC
		LIMIT 10
	`

	rows, err := p.pool.Query(ctx, query, thresholdSecs)
	if err != nil {
		return nil, fmt.Errorf("failed to query idle transactions: %w", err)
	}
	defer rows.Close()

	var transactions []IdleTransaction
	for rows.Next() {
		var t IdleTransaction
		if err := rows.Scan(&t.PID, &t.Username, &t.DatabaseName, &t.Query, &t.IdleDurationSecs); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}
