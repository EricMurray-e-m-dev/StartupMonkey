package unit

import (
	"context"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
)

func setupTestClient(t *testing.T) *redis.Client {
	client, err := redis.NewClient("localhost:6379", "", 1) // Use DB 1 for testing
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	return client
}

func TestRegisterDetection(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	detection := &models.Detection{
		ID:         "test-det-001",
		Key:        "testdb:query:users:email:seq_scans",
		State:      models.StateActive,
		Severity:   "critical",
		Category:   "query",
		DatabaseID: "testdb",
		Value:      1000,
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}

	// Register detection
	err := client.RegisterDetection(ctx, detection)
	if err != nil {
		t.Fatalf("Failed to register detection: %v", err)
	}

	// Verify it was stored
	retrieved, err := client.GetDetection(ctx, detection.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve detection: %v", err)
	}

	if retrieved.ID != detection.ID {
		t.Errorf("Expected ID %s, got %s", detection.ID, retrieved.ID)
	}

	if retrieved.Key != detection.Key {
		t.Errorf("Expected Key %s, got %s", detection.Key, retrieved.Key)
	}

	// Clean up
	client.GetClient().Del(ctx, "detection:"+detection.ID)
	client.GetClient().Del(ctx, "detection_key:"+detection.Key)
	client.GetClient().Del(ctx, "detections:active:"+detection.DatabaseID)
}

func TestIsDetectionActive(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	detection := &models.Detection{
		ID:         "test-det-002",
		Key:        "testdb:cache:::cache_hit_rate",
		State:      models.StateActive,
		Severity:   "warning",
		Category:   "cache",
		DatabaseID: "testdb",
		Value:      0.05,
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}

	// Register detection
	client.RegisterDetection(ctx, detection)

	// Check if active
	isActive, err := client.IsDetectionActive(ctx, detection.Key)
	if err != nil {
		t.Fatalf("Failed to check if detection active: %v", err)
	}

	if !isActive {
		t.Errorf("Expected detection to be active")
	}

	// Check non-existent key
	isActive, err = client.IsDetectionActive(ctx, "nonexistent:key")
	if err != nil {
		t.Fatalf("Failed to check non-existent key: %v", err)
	}

	if isActive {
		t.Errorf("Expected non-existent detection to be inactive")
	}

	// Clean up
	client.GetClient().Del(ctx, "detection:"+detection.ID)
	client.GetClient().Del(ctx, "detection_key:"+detection.Key)
	client.GetClient().Del(ctx, "detections:active:"+detection.DatabaseID)
}

func TestMarkDetectionResolved(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	detection := &models.Detection{
		ID:         "test-det-003",
		Key:        "testdb:query:posts:user_id:seq_scans",
		State:      models.StateActive,
		Severity:   "critical",
		Category:   "query",
		DatabaseID: "testdb",
		Value:      5000,
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}

	// Register detection
	client.RegisterDetection(ctx, detection)

	// Mark as resolved
	err := client.MarkDetectionResolved(ctx, detection.ID, "index_created:posts_user_id_idx")
	if err != nil {
		t.Fatalf("Failed to mark detection resolved: %v", err)
	}

	// Verify state changed
	retrieved, err := client.GetDetection(ctx, detection.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve detection: %v", err)
	}

	if retrieved.State != models.StateResolved {
		t.Errorf("Expected state %s, got %s", models.StateResolved, retrieved.State)
	}

	if retrieved.ResolvedBy != "index_created:posts_user_id_idx" {
		t.Errorf("Expected ResolvedBy 'index_created:posts_user_id_idx', got %s", retrieved.ResolvedBy)
	}

	// Check it's no longer active
	isActive, err := client.IsDetectionActive(ctx, detection.Key)
	if err != nil {
		t.Fatalf("Failed to check if detection active: %v", err)
	}

	if isActive {
		t.Errorf("Expected resolved detection to be inactive")
	}

	// Clean up (it will auto-expire in 5 minutes, but clean now for test)
	client.GetClient().Del(ctx, "detection:"+detection.ID)
	client.GetClient().Del(ctx, "detection_key:"+detection.Key)
}

func TestGetActiveDetections(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	dbID := "testdb"

	// Register multiple detections
	detections := []*models.Detection{
		{
			ID:         "test-det-004",
			Key:        "testdb:query:users:email:seq_scans",
			State:      models.StateActive,
			Category:   "query",
			DatabaseID: dbID,
			CreatedAt:  time.Now(),
			LastSeen:   time.Now(),
		},
		{
			ID:         "test-det-005",
			Key:        "testdb:cache:::cache_hit_rate",
			State:      models.StateActive,
			Category:   "cache",
			DatabaseID: dbID,
			CreatedAt:  time.Now(),
			LastSeen:   time.Now(),
		},
	}

	for _, det := range detections {
		client.RegisterDetection(ctx, det)
	}

	// Get active detections
	active, err := client.GetActiveDetections(ctx, dbID)
	if err != nil {
		t.Fatalf("Failed to get active detections: %v", err)
	}

	if len(active) != 2 {
		t.Errorf("Expected 2 active detections, got %d", len(active))
	}

	// Clean up
	for _, det := range detections {
		client.GetClient().Del(ctx, "detection:"+det.ID)
		client.GetClient().Del(ctx, "detection_key:"+det.Key)
	}
	client.GetClient().Del(ctx, "detections:active:"+dbID)
}
