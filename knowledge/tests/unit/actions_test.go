package unit

import (
	"context"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
)

func TestRegisterAction(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	action := &models.Action{
		ID:          "test-action-001",
		DetectionID: "test-det-001",
		ActionType:  "create_index",
		DatabaseID:  "testdb",
		Status:      models.StatusQueued,
		Message:     "Action queued",
		CreatedAt:   time.Now(),
	}

	// Register action
	err := client.RegisterAction(ctx, action)
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Verify it was stored
	retrieved, err := client.GetAction(ctx, action.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve action: %v", err)
	}

	if retrieved.ID != action.ID {
		t.Errorf("Expected ID %s, got %s", action.ID, retrieved.ID)
	}

	if retrieved.ActionType != action.ActionType {
		t.Errorf("Expected ActionType %s, got %s", action.ActionType, retrieved.ActionType)
	}

	// Clean up
	client.GetClient().Del(ctx, "action:"+action.ID)
	client.GetClient().Del(ctx, "actions:database:"+action.DatabaseID)
	client.GetClient().Del(ctx, "actions:status:queued")
}

func TestUpdateActionStatus(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	action := &models.Action{
		ID:          "test-action-002",
		DetectionID: "test-det-002",
		ActionType:  "create_index",
		DatabaseID:  "testdb",
		Status:      models.StatusQueued,
		Message:     "Action queued",
		CreatedAt:   time.Now(),
	}

	// Register action
	client.RegisterAction(ctx, action)

	// Update status to executing
	err := client.UpdateActionStatus(ctx, action.ID, models.StatusExecuting, "Executing action", "")
	if err != nil {
		t.Fatalf("Failed to update action status: %v", err)
	}

	// Verify status updated
	retrieved, err := client.GetAction(ctx, action.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve action: %v", err)
	}

	if retrieved.Status != models.StatusExecuting {
		t.Errorf("Expected status %s, got %s", models.StatusExecuting, retrieved.Status)
	}

	if retrieved.StartedAt == nil {
		t.Errorf("Expected StartedAt to be set")
	}

	// Clean up
	client.GetClient().Del(ctx, "action:"+action.ID)
	client.GetClient().Del(ctx, "actions:database:"+action.DatabaseID)
	client.GetClient().Del(ctx, "actions:status:queued")
	client.GetClient().Del(ctx, "actions:status:executing")
}

func TestGetPendingActions(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	dbID := "testdb"

	// Register multiple actions with different statuses
	actions := []*models.Action{
		{
			ID:          "test-action-003",
			DetectionID: "test-det-003",
			ActionType:  "create_index",
			DatabaseID:  dbID,
			Status:      models.StatusQueued,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-action-004",
			DetectionID: "test-det-004",
			ActionType:  "deploy_pgbouncer",
			DatabaseID:  dbID,
			Status:      models.StatusExecuting,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-action-005",
			DetectionID: "test-det-005",
			ActionType:  "create_index",
			DatabaseID:  dbID,
			Status:      models.StatusCompleted,
			CreatedAt:   time.Now(),
		},
	}

	for _, a := range actions {
		client.RegisterAction(ctx, a)
	}

	// Get pending actions (should only return queued + executing)
	pending, err := client.GetPendingActions(ctx, dbID)
	if err != nil {
		t.Fatalf("Failed to get pending actions: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending actions, got %d", len(pending))
	}

	// Verify completed action not included
	for _, a := range pending {
		if a.Status == models.StatusCompleted {
			t.Errorf("Completed action should not be in pending list")
		}
	}

	// Clean up
	for _, a := range actions {
		client.GetClient().Del(ctx, "action:"+a.ID)
	}
	client.GetClient().Del(ctx, "actions:database:"+dbID)
	client.GetClient().Del(ctx, "actions:status:queued")
	client.GetClient().Del(ctx, "actions:status:executing")
	client.GetClient().Del(ctx, "actions:status:completed")
}
