package actions

import (
	"context"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type FutureFix struct {
	metadata   *models.ActionMetadata
	actionType string
	reason     string
}

func NewFutureFixAction(id, actionType, dbID, reason string) *FutureFix {
	return &FutureFix{
		metadata: &models.ActionMetadata{
			ActionID:   id,
			ActionType: actionType,
			DatabaseID: dbID,
			CreatedAt:  time.Time{},
		},
		actionType: actionType,
		reason:     reason,
	}
}

func (a *FutureFix) Execute(ctx context.Context) (*models.ActionResult, error) {
	return &models.ActionResult{
		ActionID:   a.metadata.ActionID,
		ActionType: a.metadata.ActionType,
		DatabaseID: a.metadata.DatabaseID,
		Status:     models.StatusPendingImplementation,
		Message:    a.reason,
		CreatedAt:  a.metadata.CreatedAt,
	}, nil
}

func (a *FutureFix) Validate(ctx context.Context) error {
	// Always valid - it's a placeholder
	return nil
}

func (a *FutureFix) Rollback(ctx context.Context) error {
	// Nothing to rollback
	return nil
}

func (a *FutureFix) GetMetadata() *models.ActionMetadata {
	return a.metadata
}
