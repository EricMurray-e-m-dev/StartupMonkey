package database

import (
	"context"
	"fmt"
)

type DatabaseAdapter interface {
	CreateIndex(ctx context.Context, params IndexParams) error
	DropIndex(ctx context.Context, indexName string) error
	IndexExists(ctx context.Context, indexName string) (bool, error)
	GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error)
	SetConfig(ctx context.Context, changes map[string]string) error
	GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]SlowQuery, error)
	GetCapabilities() Capabilities
	Close() error
}

type SlowQuery struct {
	QueryPattern    string  `json:"query_pattern"`
	ExecutionTimeMs float64 `json:"execution_time_ms"`
	CallCount       int32   `json:"call_count"`
	IssueType       string  `json:"issue_type"`
	Recommendation  string  `json:"recommendation"`
}

type IndexParams struct {
	TableName   string   `json:"table_name"`
	ColumnNames []string `json:"column_names"`
	IndexName   string   `json:"index_name"`
	Unique      bool     `json:"unique"`
	Concurrent  bool     `json:"concurrent"`
}

type Capabilities struct {
	SupportsIndexes              bool `json:"supports_indexes"`
	SupportsConcurrentIndexes    bool `json:"supports_concurrent_indexes"`
	SupportsUniqueIndex          bool `json:"supports_unique_index"`
	SupportsMultiColumnIndex     bool `json:"supports_multi_column_index"`
	SupportsConfigTuning         bool `json:"supports_config_tuning"`
	SupportsRuntimeConfigChanges bool `json:"supports_runtime_config_changes"`
}

var (
	ErrActionNotSupported = fmt.Errorf("action not supported by this database")
	ErrIndexAlreadyExists = fmt.Errorf("index already exists")
)
