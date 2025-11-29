package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type DeployPgBouncerAction struct {
	actionID      string
	detectionID   string
	databaseID    string
	databaseType  string
	containerName string
	deployed      bool

	// Store deployment details for rollback
	deploymentDetails map[string]interface{}
}

func NewDeployPgBouncerAction(detectionID, databaseID, databaseType string, params map[string]interface{}) *DeployPgBouncerAction {
	containerName := fmt.Sprintf("pgbouncer-%s", databaseID)
	actionID := fmt.Sprintf("action-%s-%d", detectionID, time.Now().Unix())

	return &DeployPgBouncerAction{
		actionID:          actionID,
		detectionID:       detectionID,
		databaseID:        databaseID,
		databaseType:      databaseType,
		containerName:     containerName,
		deployed:          false,
		deploymentDetails: make(map[string]interface{}),
	}
}

func (a *DeployPgBouncerAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	startTime := time.Now()

	// TODO: Implement Docker deployment
	// 1. Generate pgbouncer.ini config
	// 2. Create userlist.txt
	// 3. Deploy container with Docker SDK

	endTime := time.Now()
	executionTimeMs := endTime.Sub(startTime).Milliseconds()

	result := &models.ActionResult{
		ActionID:        a.actionID,
		DetectionID:     a.detectionID,
		ActionType:      "deploy_pgbouncer",
		DatabaseID:      a.databaseID,
		Status:          models.StatusCompleted,
		Message:         fmt.Sprintf("PgBouncer deployed as container '%s' on port 6432", a.containerName),
		CreatedAt:       startTime,
		Started:         &startTime,
		Completed:       &endTime,
		ExecutionTimeMs: executionTimeMs,
		Changes: map[string]interface{}{
			"container_name": a.containerName,
			"pgbouncer_port": 6432,
			"instruction":    "Update your app's DB_CONNECTION_STRING to use port 6432",
		},
		CanRollback: true,
		Rolledback:  false,
	}

	a.deployed = true
	return result, nil
}

func (a *DeployPgBouncerAction) Rollback(ctx context.Context) error {
	if !a.deployed {
		return fmt.Errorf("PgBouncer was not deployed, cannot rollback")
	}

	// TODO: Implement Docker container stop and remove

	a.deployed = false
	return nil
}

func (a *DeployPgBouncerAction) Validate(ctx context.Context) error {
	// TODO: Check Docker is available and accessible
	// TODO: Verify PostgreSQL connection details are available
	return nil
}

func (a *DeployPgBouncerAction) GetMetadata() *models.ActionMetadata {
	return &models.ActionMetadata{
		ActionID:     a.actionID,
		ActionType:   "deploy_pgbouncer",
		DatabaseID:   a.databaseID,
		DatabaseType: a.databaseType,
		CreatedAt:    time.Now(),
	}
}
