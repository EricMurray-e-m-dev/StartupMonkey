package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLAdapter struct {
	db           *sql.DB
	databaseName string
}

func NewMySQLAdapter(ctx context.Context, connectionString, databaseName string) (*MySQLAdapter, error) {
	// Convert connection string format if needed
	dsn := convertMySQLConnectionString(connectionString)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return &MySQLAdapter{
		db:           db,
		databaseName: databaseName,
	}, nil
}

// convertMySQLConnectionString converts URL format to MySQL DSN format.
func convertMySQLConnectionString(connStr string) string {
	if strings.HasPrefix(connStr, "mysql://") {
		connStr = strings.TrimPrefix(connStr, "mysql://")

		atIdx := strings.LastIndex(connStr, "@")
		if atIdx == -1 {
			return connStr
		}

		userPass := connStr[:atIdx]
		hostDBPart := connStr[atIdx+1:]

		slashIdx := strings.Index(hostDBPart, "/")
		if slashIdx == -1 {
			return fmt.Sprintf("%s@tcp(%s)/", userPass, hostDBPart)
		}

		hostPort := hostDBPart[:slashIdx]
		dbName := hostDBPart[slashIdx+1:]

		if qIdx := strings.Index(dbName, "?"); qIdx != -1 {
			dbName = dbName[:qIdx]
		}

		return fmt.Sprintf("%s@tcp(%s)/%s?parseTime=true", userPass, hostPort, dbName)
	}

	return connStr
}

func (m *MySQLAdapter) CreateIndex(ctx context.Context, params IndexParams) error {
	exists, err := m.IndexExists(ctx, params.IndexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return ErrIndexAlreadyExists
	}

	columns := strings.Join(params.ColumnNames, ", ")

	indexType := "INDEX"
	if params.Unique {
		indexType = "UNIQUE INDEX"
	}

	// MySQL doesn't support CONCURRENTLY, but we can use ALGORITHM=INPLACE LOCK=NONE for online DDL
	query := fmt.Sprintf(
		"ALTER TABLE %s ADD %s %s (%s) ALGORITHM=INPLACE, LOCK=NONE",
		params.TableName, indexType, params.IndexName, columns,
	)

	_, err = m.db.ExecContext(ctx, query)
	if err != nil {
		// Fallback to standard CREATE INDEX if online DDL fails
		query = fmt.Sprintf(
			"CREATE %s %s ON %s (%s)",
			indexType, params.IndexName, params.TableName, columns,
		)
		_, err = m.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (m *MySQLAdapter) DropIndex(ctx context.Context, indexName string) error {
	// Need to find which table the index belongs to
	tableName, err := m.getTableForIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to find table for index: %w", err)
	}

	if tableName == "" {
		// Index doesn't exist, nothing to do
		return nil
	}

	query := fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)

	_, err = m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

func (m *MySQLAdapter) getTableForIndex(ctx context.Context, indexName string) (string, error) {
	query := `
		SELECT TABLE_NAME 
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND INDEX_NAME = ?
		LIMIT 1
	`

	var tableName string
	err := m.db.QueryRowContext(ctx, query, indexName).Scan(&tableName)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return tableName, nil
}

func (m *MySQLAdapter) IndexExists(ctx context.Context, indexName string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND INDEX_NAME = ?
	`

	var count int
	err := m.db.QueryRowContext(ctx, query, indexName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}

	return count > 0, nil
}

func (m *MySQLAdapter) GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error) {
	config := make(map[string]string)

	for _, param := range parameters {
		var varName, value string
		query := fmt.Sprintf("SHOW VARIABLES LIKE '%s'", param)
		err := m.db.QueryRowContext(ctx, query).Scan(&varName, &value)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to get config for %s: %w", param, err)
		}
		config[param] = value
	}

	return config, nil
}

func (m *MySQLAdapter) SetConfig(ctx context.Context, changes map[string]string) error {
	for param, value := range changes {
		query := fmt.Sprintf("SET GLOBAL %s = '%s'", param, value)
		_, err := m.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to set %s to %s: %w", param, value, err)
		}
	}

	return nil
}

func (m *MySQLAdapter) GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]SlowQuery, error) {
	// Query from performance_schema if available
	query := `
		SELECT 
			DIGEST_TEXT,
			AVG_TIMER_WAIT / 1000000000 as avg_time_ms,
			COUNT_STAR as calls
		FROM performance_schema.events_statements_summary_by_digest
		WHERE SCHEMA_NAME = DATABASE()
		AND AVG_TIMER_WAIT / 1000000000 > ?
		ORDER BY AVG_TIMER_WAIT DESC
		LIMIT ?
	`

	rows, err := m.db.QueryContext(ctx, query, thresholdMs, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query slow queries: %w", err)
	}
	defer rows.Close()

	var slowQueries []SlowQuery

	for rows.Next() {
		var rawQuery sql.NullString
		var execTime float64
		var calls int32

		if err := rows.Scan(&rawQuery, &execTime, &calls); err != nil {
			return nil, fmt.Errorf("failed to scan slow queries: %w", err)
		}

		queryStr := ""
		if rawQuery.Valid {
			queryStr = sanitiseMySQLQuery(rawQuery.String)
		}

		issueType, recommendation := analyseMySQLQuery(rawQuery.String)

		slowQueries = append(slowQueries, SlowQuery{
			QueryPattern:    queryStr,
			ExecutionTimeMs: execTime,
			CallCount:       calls,
			IssueType:       issueType,
			Recommendation:  recommendation,
		})
	}

	return slowQueries, nil
}

func (m *MySQLAdapter) VacuumTable(ctx context.Context, tableName string) error {
	// MySQL uses OPTIMIZE TABLE instead of VACUUM
	query := fmt.Sprintf("OPTIMIZE TABLE %s", tableName)

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to optimize table %s: %w", tableName, err)
	}

	return nil
}

func (m *MySQLAdapter) GetDeadTuples(ctx context.Context, tableName string) (int64, error) {
	// MySQL doesn't track dead tuples the same way as PostgreSQL
	// We can check table fragmentation instead
	query := `
		SELECT DATA_FREE 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ?
	`

	var dataFree int64
	err := m.db.QueryRowContext(ctx, query, tableName).Scan(&dataFree)
	if err != nil {
		return 0, fmt.Errorf("failed to get fragmentation for %s: %w", tableName, err)
	}

	return dataFree, nil
}

func (m *MySQLAdapter) TerminateQuery(ctx context.Context, pid int32, graceful bool) error {
	// MySQL uses KILL command
	// KILL QUERY only terminates the query, KILL terminates the connection
	var query string
	if graceful {
		query = fmt.Sprintf("KILL QUERY %d", pid)
	} else {
		query = fmt.Sprintf("KILL %d", pid)
	}

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to terminate query: %w", err)
	}

	return nil
}

func (m *MySQLAdapter) GetCapabilities() Capabilities {
	return Capabilities{
		SupportsIndexes:              true,
		SupportsConcurrentIndexes:    false, // MySQL doesn't have CONCURRENTLY
		SupportsUniqueIndex:          true,
		SupportsMultiColumnIndex:     true,
		SupportsConfigTuning:         true,
		SupportsRuntimeConfigChanges: true,
		SupportsVacuum:               true, // Via OPTIMIZE TABLE
		SupportsQueryTermination:     true,
	}
}

func (m *MySQLAdapter) Close() error {
	if m.db != nil {
		err := m.db.Close()
		m.db = nil
		return err
	}
	return nil
}

// Helper: Sanitise MySQL queries
func sanitiseMySQLQuery(query string) string {
	query = regexp.MustCompile(`'[^']*'`).ReplaceAllString(query, "?")
	query = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(query, "?")

	if len(query) > 200 {
		query = query[:200] + "..."
	}

	query = strings.Join(strings.Fields(query), " ")

	return query
}

// Helper: Analyse MySQL query to determine issue type
func analyseMySQLQuery(query string) (issueType string, recommendation string) {
	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "full table scan") || strings.Contains(queryLower, "using where") {
		return "sequential_scan", "Consider adding an index on the filtered columns"
	}

	if strings.Contains(queryLower, "order by") && !strings.Contains(queryLower, "using index") {
		return "missing_index", "Add index on ORDER BY columns for faster sorting"
	}

	if strings.Count(queryLower, "join") >= 3 {
		return "complex_join", "Consider simplifying joins or adding indexes on join columns"
	}

	if strings.Contains(queryLower, "select *") {
		return "inefficient_select", "Select only required columns instead of SELECT *"
	}

	return "high_latency", "Review query execution plan with EXPLAIN"
}
