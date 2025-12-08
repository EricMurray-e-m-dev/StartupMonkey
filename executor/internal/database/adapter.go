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
	QueryPattern    string
	ExecutionTimeMs float64
	CallCount       int32
	IssueType       string
	Recommendation  string
}

type IndexParams struct {
	TableName   string
	ColumnNames []string
	IndexName   string
	Unique      bool
	Concurrent  bool
}

type Capabilities struct {
	SupportsIndexes              bool
	SupportsConcurrentIndexes    bool
	SupportsUniqueIndex          bool
	SupportsMultiColumnIndex     bool
	SupportsConfigTuning         bool
	SupportsRuntimeConfigChanges bool
}

var (
	ErrActionNotSupported = fmt.Errorf("action not supported by this database")
	ErrIndexAlreadyExists = fmt.Errorf("index already exists")
)
