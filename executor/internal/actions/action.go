package actions

import (
	"context"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type Action interface {
	Execute(ctx context.Context) (*models.ActionResult, error)
	Rollback(ctx context.Context) error
	Validate(ctx context.Context) error
	GetMetadata() *models.ActionMetadata
}
