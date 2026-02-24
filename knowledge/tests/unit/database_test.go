package unit

import (
	"context"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
)

func TestRegisterDatabase(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	database := &models.Database{
		ID:               "test-db-001",
		ConnectionString: "postgresql://user:pass@localhost:5432/testdb",
		DatabaseType:     "postgres",
		DatabaseName:     "Test Database",
		Host:             "localhost",
		Port:             5432,
		RegisteredAt:     time.Now(),
		LastSeen:         time.Now(),
		Status:           "healthy",
		HealthScore:      1.0,
		Enabled:          true,
	}

	err := client.RegisterDatabase(ctx, database)
	if err != nil {
		t.Fatalf("Failed to register database: %v", err)
	}

	retrieved, err := client.GetDatabase(ctx, database.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve database: %v", err)
	}

	if retrieved.ID != database.ID {
		t.Errorf("Expected ID %s, got %s", database.ID, retrieved.ID)
	}

	if retrieved.ConnectionString != database.ConnectionString {
		t.Errorf("Expected ConnectionString %s, got %s", database.ConnectionString, retrieved.ConnectionString)
	}

	if retrieved.Enabled != database.Enabled {
		t.Errorf("Expected Enabled %v, got %v", database.Enabled, retrieved.Enabled)
	}

	// Clean up
	client.GetClient().Del(ctx, "database:"+database.ID)
	client.GetClient().SRem(ctx, "databases:all", database.ID)
}

func TestListDatabases(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	databases := []*models.Database{
		{
			ID:               "test-db-002",
			ConnectionString: "postgresql://localhost:5432/db1",
			DatabaseType:     "postgres",
			DatabaseName:     "Database 1",
			Status:           "healthy",
			HealthScore:      1.0,
			Enabled:          true,
			RegisteredAt:     time.Now(),
			LastSeen:         time.Now(),
		},
		{
			ID:               "test-db-003",
			ConnectionString: "postgresql://localhost:5432/db2",
			DatabaseType:     "postgres",
			DatabaseName:     "Database 2",
			Status:           "healthy",
			HealthScore:      0.8,
			Enabled:          false,
			RegisteredAt:     time.Now(),
			LastSeen:         time.Now(),
		},
	}

	for _, db := range databases {
		client.RegisterDatabase(ctx, db)
	}

	listed, err := client.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("Failed to list databases: %v", err)
	}

	if len(listed) < 2 {
		t.Errorf("Expected at least 2 databases, got %d", len(listed))
	}

	// Clean up
	for _, db := range databases {
		client.GetClient().Del(ctx, "database:"+db.ID)
		client.GetClient().SRem(ctx, "databases:all", db.ID)
	}
}

func TestUpdateDatabaseHealth(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	database := &models.Database{
		ID:           "test-db-004",
		DatabaseType: "postgres",
		DatabaseName: "Health Test DB",
		Status:       "healthy",
		HealthScore:  1.0,
		Enabled:      true,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}

	client.RegisterDatabase(ctx, database)

	// Update health to degraded
	newLastSeen := time.Now().Unix()
	err := client.UpdateDatabaseHealth(ctx, database.ID, newLastSeen, "degraded", 0.6)
	if err != nil {
		t.Fatalf("Failed to update database health: %v", err)
	}

	retrieved, err := client.GetDatabase(ctx, database.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve database: %v", err)
	}

	if retrieved.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got %s", retrieved.Status)
	}

	if retrieved.HealthScore != 0.6 {
		t.Errorf("Expected HealthScore 0.6, got %f", retrieved.HealthScore)
	}

	// Clean up
	client.GetClient().Del(ctx, "database:"+database.ID)
	client.GetClient().SRem(ctx, "databases:all", database.ID)
}

func TestUnregisterDatabase(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	database := &models.Database{
		ID:           "test-db-005",
		DatabaseType: "postgres",
		DatabaseName: "Unregister Test DB",
		Status:       "healthy",
		Enabled:      true,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}

	client.RegisterDatabase(ctx, database)

	// Verify it exists
	_, err := client.GetDatabase(ctx, database.ID)
	if err != nil {
		t.Fatalf("Database should exist before unregister: %v", err)
	}

	// Unregister
	err = client.UnregisterDatabase(ctx, database.ID)
	if err != nil {
		t.Fatalf("Failed to unregister database: %v", err)
	}

	// Verify it's gone
	_, err = client.GetDatabase(ctx, database.ID)
	if err == nil {
		t.Errorf("Database should not exist after unregister")
	}
}

func TestGetDatabaseNotFound(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	_, err := client.GetDatabase(ctx, "nonexistent-db")
	if err == nil {
		t.Errorf("Expected error for nonexistent database")
	}
}
