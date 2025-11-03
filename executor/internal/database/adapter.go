package database

import (
	"context"
	"fmt"
)

type DatabaseAdapter interface {
	CreateIndex(ctx context.Context, params IndexParams) error
	DropIndex(ctx context.Context, indexName string) error
	IndexExists(ctx context.Context, indexName string) (bool, error)
	GetCapabilities() Capabilities
	Close() error
}

type IndexParams struct {
	TableName   string
	ColumnNames []string
	IndexName   string
	Unique      bool
	Concurrent  bool
}

type Capabilities struct {
	SupportsIndexes           bool
	SupportsConcurrentIndexes bool
	SupportsUniqueIndex       bool
	SupportsMultiColumnIndex  bool
}

var (
	ErrActionNotSupported = fmt.Errorf("action not supported by this database")
	ErrIndexAlreadyExists = fmt.Errorf("index already exists")
)
