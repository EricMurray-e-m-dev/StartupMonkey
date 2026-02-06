package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type TerminateQueryAction struct {
	metadata *models.ActionMetadata
	adapter  database.DatabaseAdapter
	pid      int32
	username string
	graceful bool
}

func NewTerminateQueryAction(
	metadata *models.ActionMetadata,
	adapter database.DatabaseAdapter,
	pid int32,
	username string,
	graceful bool,
) *TerminateQueryAction {
	return &TerminateQueryAction{
		metadata: metadata,
		adapter:  adapter,
		pid:      pid,
		username: username,
		graceful: graceful,
	}
}

func (a *TerminateQueryAction) GetMetadata() *models.ActionMetadata {
	return a.metadata
}

func (a *TerminateQueryAction) Validate(ctx context.Context) error {
	caps := a.adapter.GetCapabilities()
	if !caps.SupportsQueryTermination {
		return database.ErrActionNotSupported
	}

	if a.pid <= 0 {
		return fmt.Errorf("invalid PID: %d", a.pid)
	}

	return nil
}

func (a *TerminateQueryAction) Execute(ctx context.Context) (*models.ActionResult, error) {
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

	method := "pg_cancel_backend"
	if !a.graceful {
		method = "pg_terminate_backend"
	}

	err := a.adapter.TerminateQuery(ctx, a.pid, a.graceful)
	if err != nil {
		// If graceful failed, try forceful
		if a.graceful {
			err = a.adapter.TerminateQuery(ctx, a.pid, false)
			if err == nil {
				method = "pg_terminate_backend (fallback)"
			}
		}

		if err != nil {
			return &models.ActionResult{
				ActionID:        a.metadata.ActionID,
				ActionType:      a.metadata.ActionType,
				DatabaseID:      a.metadata.DatabaseID,
				Status:          models.StatusFailed,
				Message:         "Query termination failed",
				Error:           err.Error(),
				CreatedAt:       a.metadata.CreatedAt,
				Started:         &started,
				ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
				CanRollback:     false,
			}, nil
		}
	}

	completed := time.Now()

	return &models.ActionResult{
		ActionID:        a.metadata.ActionID,
		ActionType:      a.metadata.ActionType,
		DatabaseID:      a.metadata.DatabaseID,
		Status:          models.StatusCompleted,
		Message:         fmt.Sprintf("Query terminated (PID %d)", a.pid),
		CreatedAt:       a.metadata.CreatedAt,
		Started:         &started,
		Completed:       &completed,
		ExecutionTimeMs: int64(time.Since(startTime).Milliseconds()),
		Changes: map[string]interface{}{
			"pid":      a.pid,
			"username": a.username,
			"method":   method,
			"graceful": a.graceful,
		},
		CanRollback: false, // Cannot un-terminate a query
	}, nil
}

func (a *TerminateQueryAction) Rollback(ctx context.Context) error {
	// Cannot rollback query termination
	return nil
}
