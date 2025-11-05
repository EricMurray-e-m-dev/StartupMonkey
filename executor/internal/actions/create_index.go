package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type CreateIndexAction struct {
	metadata     *models.ActionMetadata
	adapter      database.DatabaseAdapter
	tableName    string
	columnNames  []string
	indexName    string
	unique       bool
	indexCreated bool
}

func NewCreateIndexAction(metadata *models.ActionMetadata, adapter database.DatabaseAdapter, tableName string, columnNames []string, unique bool) *CreateIndexAction {
	indexName := fmt.Sprintf("idx_%s_%s_%s", metadata.DatabaseID, tableName, columnNames[0])

	return &CreateIndexAction{
		metadata:    metadata,
		adapter:     adapter,
		tableName:   tableName,
		columnNames: columnNames,
		indexName:   indexName,
		unique:      unique,
	}
}

func (a *CreateIndexAction) GetMetadata() *models.ActionMetadata {
	return a.metadata
}

func (a *CreateIndexAction) Validate(ctx context.Context) error {
	caps := a.adapter.GetCapabilities()
	if !caps.SupportsIndexes {
		return database.ErrActionNotSupported
	}

	if len(a.columnNames) > 1 && !caps.SupportsMultiColumnIndex {
		return database.ErrActionNotSupported
	}

	if a.unique && !caps.SupportsUniqueIndex {
		return database.ErrActionNotSupported
	}

	exists, err := a.adapter.IndexExists(ctx, a.indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existance: %w", err)
	}

	if exists {
		return database.ErrIndexAlreadyExists
	}

	return nil
}

func (a *CreateIndexAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	startTime := time.Now()

	started := time.Now()

	if err := a.Validate(ctx); err != nil {
		return &models.ActionResult{
			ActionID:        a.metadata.ActionID,
			DetectionID:     "",
			ActionType:      a.metadata.ActionType,
			DatabaseID:      a.metadata.DatabaseID,
			Status:          models.StatusFailed,
			Message:         "Validation error",
			Error:           err.Error(),
			CreatedAt:       a.metadata.CreatedAt,
			Started:         &started,
			Completed:       nil,
			ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
			CanRollback:     false,
			Rolledback:      false,
		}, nil
	}

	caps := a.adapter.GetCapabilities()
	params := database.IndexParams{
		TableName:   a.tableName,
		ColumnNames: a.columnNames,
		IndexName:   a.indexName,
		Unique:      a.unique,
		Concurrent:  caps.SupportsConcurrentIndexes,
	}

	err := a.adapter.CreateIndex(ctx, params)
	if err != nil {
		return &models.ActionResult{
			ActionID:        a.metadata.ActionID,
			ActionType:      a.metadata.ActionType,
			DatabaseID:      a.metadata.DatabaseID,
			Status:          models.StatusFailed,
			Message:         "Index creation failed",
			Error:           err.Error(),
			CreatedAt:       a.metadata.CreatedAt,
			Started:         &started,
			ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
			CanRollback:     false,
		}, nil
	}

	a.indexCreated = true

	completed := time.Now()
	return &models.ActionResult{
		ActionID:        a.metadata.ActionID,
		ActionType:      a.metadata.ActionType,
		DatabaseID:      a.metadata.DatabaseID,
		Status:          models.StatusCompleted,
		Message:         fmt.Sprintf("Index %s created successfully", a.indexName),
		CreatedAt:       a.metadata.CreatedAt,
		Started:         &started,
		Completed:       &completed,
		ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
		Changes: map[string]interface{}{
			"index_name":   a.indexName,
			"table_name":   a.tableName,
			"column_names": a.columnNames,
			"unique":       a.unique,
			"concurrent":   params.Concurrent,
		},
		CanRollback: true,
		Rolledback:  false,
	}, nil
}

func (a *CreateIndexAction) Rollback(ctx context.Context) error {
	if !a.indexCreated {
		return nil
	}

	exists, err := a.adapter.IndexExists(ctx, a.indexName)
	if err != nil {
		return fmt.Errorf("failed to check index: %w", err)
	}

	if !exists {
		return nil
	}

	err = a.adapter.DropIndex(ctx, a.indexName)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	a.indexCreated = false
	return nil
}
