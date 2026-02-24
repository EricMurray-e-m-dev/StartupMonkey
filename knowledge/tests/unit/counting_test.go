package unit

import (
	"context"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
)

func TestCountAllActiveDetections(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Register a database first (required for counting)
	db := &models.Database{
		ID:           "test-count-db",
		DatabaseType: "postgres",
		Enabled:      true,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	client.RegisterDatabase(ctx, db)

	// Register detections
	detections := []*models.Detection{
		{
			ID:         "count-det-001",
			Key:        "test-count-db:det1",
			State:      models.StateActive,
			DatabaseID: "test-count-db",
			CreatedAt:  time.Now(),
			LastSeen:   time.Now(),
		},
		{
			ID:         "count-det-002",
			Key:        "test-count-db:det2",
			State:      models.StateActive,
			DatabaseID: "test-count-db",
			CreatedAt:  time.Now(),
			LastSeen:   time.Now(),
		},
	}

	for _, det := range detections {
		client.RegisterDetection(ctx, det)
	}

	count, err := client.CountAllActiveDetections(ctx)
	if err != nil {
		t.Fatalf("Failed to count active detections: %v", err)
	}

	if count < 2 {
		t.Errorf("Expected at least 2 active detections, got %d", count)
	}

	// Clean up
	for _, det := range detections {
		client.GetClient().Del(ctx, "detection:"+det.ID)
		client.GetClient().Del(ctx, "detection_key:"+det.Key)
	}
	client.GetClient().Del(ctx, "detections:active:test-count-db")
	client.GetClient().Del(ctx, "database:test-count-db")
	client.GetClient().SRem(ctx, "databases:all", "test-count-db")
}

func TestCountActionsByStatus(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	actions := []*models.Action{
		{
			ID:         "count-action-001",
			ActionType: "create_index",
			DatabaseID: "testdb",
			Status:     models.StatusQueued,
			CreatedAt:  time.Now(),
		},
		{
			ID:         "count-action-002",
			ActionType: "create_index",
			DatabaseID: "testdb",
			Status:     models.StatusQueued,
			CreatedAt:  time.Now(),
		},
		{
			ID:         "count-action-003",
			ActionType: "create_index",
			DatabaseID: "testdb",
			Status:     models.StatusCompleted,
			CreatedAt:  time.Now(),
		},
	}

	for _, a := range actions {
		client.RegisterAction(ctx, a)
	}

	queuedCount, err := client.CountActionsByStatus(ctx, models.StatusQueued)
	if err != nil {
		t.Fatalf("Failed to count queued actions: %v", err)
	}

	if queuedCount < 2 {
		t.Errorf("Expected at least 2 queued actions, got %d", queuedCount)
	}

	completedCount, err := client.CountActionsByStatus(ctx, models.StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to count completed actions: %v", err)
	}

	if completedCount < 1 {
		t.Errorf("Expected at least 1 completed action, got %d", completedCount)
	}

	// Clean up
	for _, a := range actions {
		client.GetClient().Del(ctx, "action:"+a.ID)
	}
	client.GetClient().Del(ctx, "actions:database:testdb")
	client.GetClient().Del(ctx, "action:status:queued")
	client.GetClient().Del(ctx, "action:status:completed")
}

func TestGetDetectionIDByKey(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	detection := &models.Detection{
		ID:         "key-lookup-det-001",
		Key:        "testdb:key:lookup:test",
		State:      models.StateActive,
		DatabaseID: "testdb",
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}

	client.RegisterDetection(ctx, detection)

	// Lookup by key
	id, err := client.GetDetectionIDByKey(ctx, detection.Key)
	if err != nil {
		t.Fatalf("Failed to get detection ID by key: %v", err)
	}

	if id != detection.ID {
		t.Errorf("Expected ID %s, got %s", detection.ID, id)
	}

	// Lookup nonexistent key
	id, err = client.GetDetectionIDByKey(ctx, "nonexistent:key")
	if err != nil {
		t.Fatalf("Unexpected error for nonexistent key: %v", err)
	}

	if id != "" {
		t.Errorf("Expected empty ID for nonexistent key, got %s", id)
	}

	// Clean up
	client.GetClient().Del(ctx, "detection:"+detection.ID)
	client.GetClient().Del(ctx, "detection_key:"+detection.Key)
	client.GetClient().Del(ctx, "detections:active:testdb")
}

func TestGetActionByStatus(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	actions := []*models.Action{
		{
			ID:         "status-action-001",
			ActionType: "create_index",
			DatabaseID: "testdb",
			Status:     models.StatusFailed,
			CreatedAt:  time.Now(),
		},
		{
			ID:         "status-action-002",
			ActionType: "vacuum_table",
			DatabaseID: "testdb",
			Status:     models.StatusFailed,
			CreatedAt:  time.Now(),
		},
	}

	for _, a := range actions {
		client.RegisterAction(ctx, a)
	}

	failed, err := client.GetActionByStatus(ctx, models.StatusFailed)
	if err != nil {
		t.Fatalf("Failed to get actions by status: %v", err)
	}

	if len(failed) < 2 {
		t.Errorf("Expected at least 2 failed actions, got %d", len(failed))
	}

	// Clean up
	for _, a := range actions {
		client.GetClient().Del(ctx, "action:"+a.ID)
	}
	client.GetClient().Del(ctx, "actions:database:testdb")
	client.GetClient().Del(ctx, "action:status:failed")
}
