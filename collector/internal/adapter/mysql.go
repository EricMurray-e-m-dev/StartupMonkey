// Package adapter provides database-specific metric collection implementations.
package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLAdapter implements MetricAdapter for MySQL databases.
type MySQLAdapter struct {
	connectionString         string
	databaseID               string
	db                       *sql.DB
	performanceSchemaEnabled bool
}

// NewMySQLAdapter creates a new MySQL adapter.
func NewMySQLAdapter(connectionString string, databaseID string) *MySQLAdapter {
	return &MySQLAdapter{
		connectionString: connectionString,
		databaseID:       databaseID,
	}
}

// Connect establishes a connection to the MySQL database.
func (m *MySQLAdapter) Connect() error {
	// Convert connection string format if needed
	// Input: mysql://user:pass@host:port/dbname
	// Required: user:pass@tcp(host:port)/dbname
	dsn := m.convertConnectionString(m.connectionString)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.db = db

	// Check if performance_schema is available
	m.checkPerformanceSchema(ctx)

	return nil
}

// convertConnectionString converts URL format to MySQL DSN format.
func (m *MySQLAdapter) convertConnectionString(connStr string) string {
	// Handle mysql://user:pass@host:port/dbname format
	if strings.HasPrefix(connStr, "mysql://") {
		connStr = strings.TrimPrefix(connStr, "mysql://")

		// Split user:pass@host:port/dbname
		atIdx := strings.LastIndex(connStr, "@")
		if atIdx == -1 {
			return connStr
		}

		userPass := connStr[:atIdx]
		hostDBPart := connStr[atIdx+1:]

		// Split host:port/dbname
		slashIdx := strings.Index(hostDBPart, "/")
		if slashIdx == -1 {
			return fmt.Sprintf("%s@tcp(%s)/", userPass, hostDBPart)
		}

		hostPort := hostDBPart[:slashIdx]
		dbName := hostDBPart[slashIdx+1:]

		// Remove query params for now, add parseTime for timestamp handling
		if qIdx := strings.Index(dbName, "?"); qIdx != -1 {
			dbName = dbName[:qIdx]
		}

		return fmt.Sprintf("%s@tcp(%s)/%s?parseTime=true", userPass, hostPort, dbName)
	}

	return connStr
}

// checkPerformanceSchema verifies if performance_schema is available.
func (m *MySQLAdapter) checkPerformanceSchema(ctx context.Context) {
	var enabled string
	err := m.db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'performance_schema'").Scan(&enabled, &enabled)
	if err != nil {
		log.Printf("Warning: could not check performance_schema: %v", err)
		m.performanceSchemaEnabled = false
		return
	}

	m.performanceSchemaEnabled = strings.ToUpper(enabled) == "ON"
	if !m.performanceSchemaEnabled {
		log.Printf("Warning: performance_schema is disabled")
	}
}

// CollectMetrics gathers metrics from the MySQL database.
func (m *MySQLAdapter) CollectMetrics() (*RawMetrics, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := NewRawMetrics(m.databaseID, "mysql")

	// Connection metrics
	activeConn, err := m.getActiveConnections(ctx)
	if err != nil {
		log.Printf("Warning: failed to get active connections: %v", err)
	}

	maxConn, err := m.getMaxConnections(ctx)
	if err != nil {
		log.Printf("Warning: failed to get max connections: %v", err)
	}

	metrics.Connections = &ConnectionMetrics{
		Active: &activeConn,
		Max:    &maxConn,
	}

	// Storage metrics
	dbSizeBytes, err := m.getDatabaseSizeBytes(ctx)
	if err != nil {
		log.Printf("Warning: failed to get database size: %v", err)
	} else {
		metrics.Storage = &StorageMetrics{
			UsedSizeBytes: &dbSizeBytes,
		}
		dbSizeMB := float64(dbSizeBytes) / (1024 * 1024)
		metrics.ExtendedMetrics["mysql.database_size_mb"] = dbSizeMB
	}

	// Cache metrics (InnoDB buffer pool)
	cacheHitRate, err := m.getCacheHitRate(ctx)
	if err != nil {
		log.Printf("Warning: failed to get cache hit rate: %v", err)
	} else {
		metrics.Cache = &CacheMetrics{
			HitRate: &cacheHitRate,
		}
	}

	// Query metrics - table scans from performance_schema
	if m.performanceSchemaEnabled {
		seqScans, err := m.getFullTableScans(ctx)
		if err != nil {
			log.Printf("Warning: failed to get table scans: %v", err)
		} else {
			metrics.Queries = &QueryMetrics{
				SequentialScans: &seqScans,
			}
		}

		// Table scan statistics
		tableStats, err := m.getTableScanStats(ctx)
		if err != nil {
			log.Printf("Warning: failed to get table scan stats: %v", err)
		} else if len(tableStats) > 0 {
			worstTable := tableStats[0]

			for _, table := range tableStats {
				prefix := fmt.Sprintf("mysql.table.%s", table.TableName)
				metrics.ExtendedMetrics[prefix+".seq_scans"] = float64(table.SeqScans)
				metrics.ExtendedMetrics[prefix+".seq_tup_read"] = float64(table.RowsRead)
				metrics.ExtendedMetrics[prefix+".idx_scans"] = float64(table.IdxScans)
			}

			metrics.Labels["mysql.worst_seq_scan_table"] = worstTable.TableName

			// For MySQL, recommend index on columns used in WHERE without index
			// This is simplified - just use the table name for now
			// A more sophisticated approach would analyse slow query log
			recommendedColumn := m.guessIndexColumn(ctx, worstTable.TableName)
			if recommendedColumn != "" {
				metrics.Labels["mysql.recommended_index_column"] = recommendedColumn
			}
		}
	}

	return metrics, nil
}

// getActiveConnections returns the number of active connections.
func (m *MySQLAdapter) getActiveConnections(ctx context.Context) (int32, error) {
	var count int32
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.processlist 
		WHERE command != 'Sleep'
	`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// getMaxConnections returns the maximum allowed connections.
func (m *MySQLAdapter) getMaxConnections(ctx context.Context) (int32, error) {
	var varName string
	var maxConn int32
	err := m.db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'max_connections'").Scan(&varName, &maxConn)
	if err != nil {
		return 0, err
	}
	return maxConn, nil
}

// getDatabaseSizeBytes returns the size of the current database in bytes.
func (m *MySQLAdapter) getDatabaseSizeBytes(ctx context.Context) (int64, error) {
	var sizeBytes int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(data_length + index_length), 0)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
	`).Scan(&sizeBytes)
	if err != nil {
		return 0, err
	}
	return sizeBytes, nil
}

// getCacheHitRate returns the InnoDB buffer pool hit rate.
func (m *MySQLAdapter) getCacheHitRate(ctx context.Context) (float64, error) {
	var readRequests, diskReads int64

	// Get buffer pool read requests (logical reads)
	var varName string
	err := m.db.QueryRowContext(ctx, "SHOW STATUS LIKE 'Innodb_buffer_pool_read_requests'").Scan(&varName, &readRequests)
	if err != nil {
		return 0, fmt.Errorf("failed to get read requests: %w", err)
	}

	// Get buffer pool reads from disk (physical reads)
	err = m.db.QueryRowContext(ctx, "SHOW STATUS LIKE 'Innodb_buffer_pool_reads'").Scan(&varName, &diskReads)
	if err != nil {
		return 0, fmt.Errorf("failed to get disk reads: %w", err)
	}

	if readRequests == 0 {
		return 1.0, nil // No reads yet, assume 100% hit rate
	}

	// Hit rate = (requests - disk reads) / requests
	hitRate := float64(readRequests-diskReads) / float64(readRequests)
	return hitRate, nil
}

// MySQLTableScanStat holds table scan statistics for MySQL.
type MySQLTableScanStat struct {
	TableName string
	SeqScans  int64 // COUNT_READ (full table scans)
	RowsRead  int64 // SUM_ROWS_FETCHED
	IdxScans  int64 // Index reads
}

// getFullTableScans returns total full table scans across all tables.
func (m *MySQLAdapter) getFullTableScans(ctx context.Context) (int32, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(COUNT_READ), 0)
		FROM performance_schema.table_io_waits_summary_by_index_usage
		WHERE INDEX_NAME IS NULL
		AND OBJECT_SCHEMA = DATABASE()
	`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// getTableScanStats returns per-table scan statistics.
func (m *MySQLAdapter) getTableScanStats(ctx context.Context) ([]MySQLTableScanStat, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT 
			t1.OBJECT_NAME as table_name,
			COALESCE(t1.COUNT_READ, 0) as full_scans,
			COALESCE(t1.SUM_ROWS_FETCHED, 0) as rows_read,
			COALESCE(SUM(t2.COUNT_READ), 0) as index_reads
		FROM performance_schema.table_io_waits_summary_by_index_usage t1
		LEFT JOIN performance_schema.table_io_waits_summary_by_index_usage t2
			ON t1.OBJECT_SCHEMA = t2.OBJECT_SCHEMA 
			AND t1.OBJECT_NAME = t2.OBJECT_NAME
			AND t2.INDEX_NAME IS NOT NULL
		WHERE t1.INDEX_NAME IS NULL
		AND t1.OBJECT_SCHEMA = DATABASE()
		AND t1.COUNT_READ > 0
		GROUP BY t1.OBJECT_NAME, t1.COUNT_READ, t1.SUM_ROWS_FETCHED
		ORDER BY t1.COUNT_READ DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []MySQLTableScanStat
	for rows.Next() {
		var s MySQLTableScanStat
		if err := rows.Scan(&s.TableName, &s.SeqScans, &s.RowsRead, &s.IdxScans); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// guessIndexColumn attempts to identify a column that should be indexed.
// This is a simplified approach - checks for columns commonly used in WHERE clauses.
func (m *MySQLAdapter) guessIndexColumn(ctx context.Context, tableName string) string {
	// Get columns that are likely filter candidates (not primary key, not already indexed)
	rows, err := m.db.QueryContext(ctx, `
		SELECT c.COLUMN_NAME
		FROM information_schema.columns c
		LEFT JOIN information_schema.statistics s
			ON c.TABLE_SCHEMA = s.TABLE_SCHEMA
			AND c.TABLE_NAME = s.TABLE_NAME
			AND c.COLUMN_NAME = s.COLUMN_NAME
		WHERE c.TABLE_SCHEMA = DATABASE()
		AND c.TABLE_NAME = ?
		AND c.COLUMN_KEY = ''
		AND s.COLUMN_NAME IS NULL
		AND (c.COLUMN_NAME LIKE '%_id' OR c.COLUMN_NAME LIKE '%_at' OR c.COLUMN_NAME = 'status')
		LIMIT 1
	`, tableName)
	if err != nil {
		return ""
	}
	defer rows.Close()

	if rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err == nil {
			return columnName
		}
	}

	return ""
}

// Close closes the database connection.
func (m *MySQLAdapter) Close() error {
	if m.db != nil {
		err := m.db.Close()
		m.db = nil
		return err
	}
	return nil
}

// HealthCheck verifies the database connection is alive.
func (m *MySQLAdapter) HealthCheck() error {
	if m.db == nil {
		return ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.db.PingContext(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// GetUnavailableFeatures returns a list of features that are not available.
func (m *MySQLAdapter) GetUnavailableFeatures() []string {
	var features []string
	if !m.performanceSchemaEnabled {
		features = append(features, "performance_schema")
	}
	return features
}
