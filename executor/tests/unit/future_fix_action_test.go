package unit

import (
	"context"
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFutureFixAction_Execute(t *testing.T) {
	action := actions.NewFutureFixAction("action-1", "some_future_action", "test-db", "Not yet implemented")

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusPendingImplementation, result.Status)
	assert.Equal(t, "Not yet implemented", result.Message)
}

func TestFutureFixAction_Validate(t *testing.T) {
	action := actions.NewFutureFixAction("action-1", "some_future_action", "test-db", "Not yet implemented")

	err := action.Validate(context.Background())

	assert.NoError(t, err)
}

func TestFutureFixAction_Rollback(t *testing.T) {
	action := actions.NewFutureFixAction("action-1", "some_future_action", "test-db", "Not yet implemented")

	err := action.Rollback(context.Background())

	assert.NoError(t, err)
}

func TestFutureFixAction_GetMetadata(t *testing.T) {
	action := actions.NewFutureFixAction("action-1", "some_future_action", "test-db", "Not yet implemented")

	metadata := action.GetMetadata()

	assert.Equal(t, "action-1", metadata.ActionID)
	assert.Equal(t, "some_future_action", metadata.ActionType)
	assert.Equal(t, "test-db", metadata.DatabaseID)
}
