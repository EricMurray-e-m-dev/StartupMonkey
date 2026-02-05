package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type VacuumTableAction struct {
	metadata  *models.ActionMetadata
	adapter   database.DatabaseAdapter
	tableName string
}

func NewVacuumTableAction(
	metadata *models.ActionMetadata,
	adapter database.DatabaseAdapter,
	tableName string,
) *VacuumTableAction {
	return &VacuumTableAction{
		metadata:  metadata,
		adapter:   adapter,
		tableName: tableName,
	}
}

func (a *VacuumTableAction) GetMetadata() *models.ActionMetadata {
	return a.metadata
}

func (a *VacuumTableAction) Validate(ctx context.Context) error {
	caps := a.adapter.GetCapabilities()
	if !caps.SupportsVacuum {
		return database.ErrActionNotSupported
	}

	if a.tableName == "" {
		return fmt.Errorf("table name is required")
	}

	return nil
}

func (a *VacuumTableAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	startTime := time.Now()
	started := time.Now()

	if err := a.Validate(ctx); err != nil {
		return &models.ActionResult{
			ActionID:        a.metadata.ActionID,
			ActionType:      a.metadata.ActionType,
			DatabaseID:      a.metadata.DatabaseID,
			Status:          models.StatusFailed,
			Message:         "Validation error",
			Error:           err.Error(),
			CreatedAt:       a.metadata.CreatedAt,
			Started:         &started,
			ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
			CanRollback:     false,
		}, nil
	}

	// Get bloat stats before vacuum
	deadTuplesBefore, err := a.adapter.GetDeadTuples(ctx, a.tableName)
	if err != nil {
		// Non-fatal, continue with vacuum
		deadTuplesBefore = -1
	}

	// Execute VACUUM ANALYZE
	err = a.adapter.VacuumTable(ctx, a.tableName)
	if err != nil {
		return &models.ActionResult{
			ActionID:        a.metadata.ActionID,
			ActionType:      a.metadata.ActionType,
			DatabaseID:      a.metadata.DatabaseID,
			Status:          models.StatusFailed,
			Message:         "VACUUM failed",
			Error:           err.Error(),
			CreatedAt:       a.metadata.CreatedAt,
			Started:         &started,
			ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
			CanRollback:     false,
		}, nil
	}

	// Get bloat stats after vacuum
	deadTuplesAfter, err := a.adapter.GetDeadTuples(ctx, a.tableName)
	if err != nil {
		deadTuplesAfter = -1
	}

	completed := time.Now()

	changes := map[string]interface{}{
		"table_name": a.tableName,
		"operation":  "VACUUM ANALYZE",
	}

	if deadTuplesBefore >= 0 {
		changes["dead_tuples_before"] = deadTuplesBefore
	}
	if deadTuplesAfter >= 0 {
		changes["dead_tuples_after"] = deadTuplesAfter
	}
	if deadTuplesBefore >= 0 && deadTuplesAfter >= 0 {
		changes["tuples_reclaimed"] = deadTuplesBefore - deadTuplesAfter
	}

	return &models.ActionResult{
		ActionID:        a.metadata.ActionID,
		ActionType:      a.metadata.ActionType,
		DatabaseID:      a.metadata.DatabaseID,
		Status:          models.StatusCompleted,
		Message:         fmt.Sprintf("VACUUM ANALYZE completed on table '%s'", a.tableName),
		CreatedAt:       a.metadata.CreatedAt,
		Started:         &started,
		Completed:       &completed,
		ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
		Changes:         changes,
		CanRollback:     false, // VACUUM is non-reversible (but also non-destructive)
	}, nil
}

func (a *VacuumTableAction) Rollback(ctx context.Context) error {
	// VACUUM cannot be rolled back, but it's also non-destructive
	// so no action needed
	return nil
}
