// Package redis provides the Redis client and data operations for the Knowledge service.
// It handles storage and retrieval of detections, actions, databases, and system configuration.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

// ===== [DETECTION OPERATIONS] =====

// RegisterDetection stores a new detection and adds it to the active set.
func (c *Client) RegisterDetection(ctx context.Context, detection *models.Detection) error {
	detectionKey := fmt.Sprintf("detection:%s", detection.ID)

	data, err := json.Marshal(detection)
	if err != nil {
		return fmt.Errorf("failed to marshal detection: %w", err)
	}

	if err := c.rdb.Set(ctx, detectionKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store detection: %w", err)
	}

	keyMapping := fmt.Sprintf("detection_key:%s", detection.Key)
	if err := c.rdb.Set(ctx, keyMapping, detection.ID, 0).Err(); err != nil {
		return fmt.Errorf("failed to store key mapping: %w", err)
	}

	activeKey := fmt.Sprintf("detections:active:%s", detection.DatabaseID)
	if err := c.rdb.SAdd(ctx, activeKey, detection.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to active set: %w", err)
	}

	return nil
}

// IsDetectionActive checks if a detection with the given key is currently active.
func (c *Client) IsDetectionActive(ctx context.Context, key string) (bool, error) {
	keyMapping := fmt.Sprintf("detection_key:%s", key)

	detectionID, err := c.rdb.Get(ctx, keyMapping).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check detection key: %w", err)
	}

	detection, err := c.GetDetection(ctx, detectionID)
	if err != nil {
		return false, err
	}

	return detection.State == models.StateActive, nil
}

// GetDetection retrieves a detection by ID.
func (c *Client) GetDetection(ctx context.Context, id string) (*models.Detection, error) {
	detectionKey := fmt.Sprintf("detection:%s", id)

	data, err := c.rdb.Get(ctx, detectionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get detection: %w", err)
	}

	var detection models.Detection
	if err := json.Unmarshal([]byte(data), &detection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal detection: %w", err)
	}

	return &detection, nil
}

// GetDetectionIDByKey retrieves the detection ID associated with a detection key.
func (c *Client) GetDetectionIDByKey(ctx context.Context, key string) (string, error) {
	keyMapping := fmt.Sprintf("detection_key:%s", key)
	id, err := c.rdb.Get(ctx, keyMapping).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return "", nil
		}
		return "", fmt.Errorf("failed to get detection ID by key: %w", err)
	}
	return id, nil
}

// MarkDetectionResolved marks a detection as resolved and sets a TTL for cleanup.
func (c *Client) MarkDetectionResolved(ctx context.Context, id string, solution string) error {
	detection, err := c.GetDetection(ctx, id)
	if err != nil {
		return err
	}

	detection.State = models.StateResolved
	detection.ResolvedBy = solution
	detection.TTL = 300

	detectionKey := fmt.Sprintf("detection:%s", detection.ID)
	data, err := json.Marshal(detection)
	if err != nil {
		return fmt.Errorf("failed to marshal detection: %w", err)
	}

	if err := c.rdb.Set(ctx, detectionKey, data, time.Duration(detection.TTL)*time.Second).Err(); err != nil {
		return fmt.Errorf("failed to update detection: %w", err)
	}

	activeKey := fmt.Sprintf("detections:active:%s", detection.DatabaseID)
	if err := c.rdb.SRem(ctx, activeKey, detection.ID).Err(); err != nil {
		return fmt.Errorf("failed to remove from active set: %w", err)
	}

	return nil
}

// GetActiveDetections retrieves all active detections for a database.
func (c *Client) GetActiveDetections(ctx context.Context, databaseID string) ([]*models.Detection, error) {
	activeKey := fmt.Sprintf("detections:active:%s", databaseID)

	detectionIDs, err := c.rdb.SMembers(ctx, activeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active detections: %w", err)
	}

	detections := make([]*models.Detection, 0, len(detectionIDs))
	for _, id := range detectionIDs {
		detection, err := c.GetDetection(ctx, id)
		if err != nil {
			continue
		}
		detections = append(detections, detection)
	}

	return detections, nil
}

// CountAllActiveDetections counts active detections across all databases.
func (c *Client) CountAllActiveDetections(ctx context.Context) (int32, error) {
	databases, err := c.ListDatabases(ctx)
	if err != nil {
		return 0, err
	}

	var total int32
	for _, db := range databases {
		activeKey := fmt.Sprintf("detections:active:%s", db.ID)
		count, err := c.rdb.SCard(ctx, activeKey).Result()
		if err != nil {
			continue
		}
		total += int32(count)
	}

	return total, nil
}

// ===== [ACTION OPERATIONS] =====

// RegisterAction stores a new action and adds it to the appropriate sets.
func (c *Client) RegisterAction(ctx context.Context, action *models.Action) error {
	actionKey := fmt.Sprintf("action:%s", action.ID)

	data, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	if err := c.rdb.Set(ctx, actionKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store action: %w", err)
	}

	dbActionsKey := fmt.Sprintf("actions:database:%s", action.DatabaseID)
	if err := c.rdb.SAdd(ctx, dbActionsKey, action.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to database set: %w", err)
	}

	statusKey := fmt.Sprintf("action:status:%s", action.Status)
	if err := c.rdb.SAdd(ctx, statusKey, action.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to status set: %w", err)
	}

	return nil
}

// UpdateActionStatus updates the status of an action and moves it between status sets.
func (c *Client) UpdateActionStatus(ctx context.Context, actionID string, status models.ActionStatus, message string, errorMsg string) error {
	action, err := c.GetAction(ctx, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action for update: %w", err)
	}

	oldStatusKey := fmt.Sprintf("action:status:%s", action.Status)
	if err := c.rdb.SRem(ctx, oldStatusKey, actionID).Err(); err != nil {
		return fmt.Errorf("failed to remove from old status set: %w", err)
	}

	action.Status = status
	action.Message = message

	if errorMsg != "" {
		action.Error = errorMsg
	}

	now := time.Now()
	switch status {
	case models.StatusExecuting:
		action.StartedAt = &now
	case models.StatusCompleted, models.StatusFailed:
		action.CompletedAt = &now
	}

	actionKey := fmt.Sprintf("action:%s", action.ID)
	data, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	if err := c.rdb.Set(ctx, actionKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to update action: %w", err)
	}

	newStatusKey := fmt.Sprintf("action:status:%s", status)
	if err := c.rdb.SAdd(ctx, newStatusKey, actionID).Err(); err != nil {
		return fmt.Errorf("failed to add to new status set: %w", err)
	}

	return nil
}

// GetAction retrieves an action by ID.
func (c *Client) GetAction(ctx context.Context, id string) (*models.Action, error) {
	actionKey := fmt.Sprintf("action:%s", id)

	data, err := c.rdb.Get(ctx, actionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	var action models.Action
	if err := json.Unmarshal([]byte(data), &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %w", err)
	}

	return &action, nil
}

// GetPendingActions retrieves all queued or executing actions for a database.
func (c *Client) GetPendingActions(ctx context.Context, databaseID string) ([]*models.Action, error) {
	dbActionsKey := fmt.Sprintf("actions:database:%s", databaseID)

	actionIDs, err := c.rdb.SMembers(ctx, dbActionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get actions for %s: %w", databaseID, err)
	}

	actions := make([]*models.Action, 0)
	for _, id := range actionIDs {
		action, err := c.GetAction(ctx, id)
		if err != nil {
			continue
		}

		if action.Status == models.StatusQueued || action.Status == models.StatusExecuting {
			actions = append(actions, action)
		}
	}

	return actions, nil
}

// GetActionByStatus retrieves all actions with a specific status.
func (c *Client) GetActionByStatus(ctx context.Context, status models.ActionStatus) ([]*models.Action, error) {
	statusKey := fmt.Sprintf("action:status:%s", status)

	actionIDs, err := c.rdb.SMembers(ctx, statusKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get actions by status: %w", err)
	}

	actions := make([]*models.Action, 0, len(actionIDs))
	for _, id := range actionIDs {
		action, err := c.GetAction(ctx, id)
		if err != nil {
			continue
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// CountActionsByStatus counts actions with a specific status.
func (c *Client) CountActionsByStatus(ctx context.Context, status models.ActionStatus) (int32, error) {
	statusKey := fmt.Sprintf("action:status:%s", status)
	count, err := c.rdb.SCard(ctx, statusKey).Result()
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// CountAllActions counts total actions across all databases.
func (c *Client) CountAllActions(ctx context.Context) (int32, error) {
	databases, err := c.ListDatabases(ctx)
	if err != nil {
		return 0, err
	}

	var total int32
	for _, db := range databases {
		dbActionsKey := fmt.Sprintf("actions:database:%s", db.ID)
		count, err := c.rdb.SCard(ctx, dbActionsKey).Result()
		if err != nil {
			continue
		}
		total += int32(count)
	}

	return total, nil
}

// ===== [DATABASE OPERATIONS] =====

// RegisterDatabase stores database connection info in Redis.
func (c *Client) RegisterDatabase(ctx context.Context, database *models.Database) error {
	databaseKey := fmt.Sprintf("database:%s", database.ID)

	data, err := json.Marshal(database)
	if err != nil {
		return fmt.Errorf("failed to marshal database: %w", err)
	}

	if err := c.rdb.Set(ctx, databaseKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store database: %w", err)
	}

	if err := c.rdb.SAdd(ctx, "databases:all", database.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to database list: %w", err)
	}

	return nil
}

// GetDatabase retrieves database connection info by ID.
func (c *Client) GetDatabase(ctx context.Context, id string) (*models.Database, error) {
	databaseKey := fmt.Sprintf("database:%s", id)

	data, err := c.rdb.Get(ctx, databaseKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var database models.Database
	if err := json.Unmarshal([]byte(data), &database); err != nil {
		return nil, fmt.Errorf("failed to unmarshal database: %w", err)
	}

	return &database, nil
}

// ListDatabases returns all registered databases.
func (c *Client) ListDatabases(ctx context.Context) ([]*models.Database, error) {
	databaseIDs, err := c.rdb.SMembers(ctx, "databases:all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get database list: %w", err)
	}

	databases := make([]*models.Database, 0, len(databaseIDs))
	for _, id := range databaseIDs {
		database, err := c.GetDatabase(ctx, id)
		if err != nil {
			continue
		}
		databases = append(databases, database)
	}

	return databases, nil
}

// UpdateDatabaseHealth updates health status and last seen timestamp.
func (c *Client) UpdateDatabaseHealth(ctx context.Context, id string, lastSeen int64, status string, healthScore float64) error {
	database, err := c.GetDatabase(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get database for update: %w", err)
	}

	database.LastSeen = time.Unix(lastSeen, 0)
	database.Status = status
	database.HealthScore = healthScore

	databaseKey := fmt.Sprintf("database:%s", id)
	data, err := json.Marshal(database)
	if err != nil {
		return fmt.Errorf("failed to marshal database: %w", err)
	}

	if err := c.rdb.Set(ctx, databaseKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	return nil
}

// UnregisterDatabase removes a database from Redis.
func (c *Client) UnregisterDatabase(ctx context.Context, id string) error {
	databaseKey := fmt.Sprintf("database:%s", id)

	if err := c.rdb.Del(ctx, databaseKey).Err(); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	if err := c.rdb.SRem(ctx, "databases:all", id).Err(); err != nil {
		return fmt.Errorf("failed to remove from database list: %w", err)
	}

	return nil
}

// ===== [CONFIGURATION OPERATIONS] =====

const systemConfigKey = "config:system"

// GetSystemConfig retrieves the system configuration from Redis.
func (c *Client) GetSystemConfig(ctx context.Context) (*pb.SystemConfig, error) {
	data, err := c.rdb.Get(ctx, systemConfigKey).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("failed to get system config: %w", err)
	}

	var config pb.SystemConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal system config: %w", err)
	}

	return &config, nil
}

// SaveSystemConfig saves the system configuration to Redis.
func (c *Client) SaveSystemConfig(ctx context.Context, config *pb.SystemConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal system config: %w", err)
	}

	if err := c.rdb.Set(ctx, systemConfigKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store system config: %w", err)
	}

	return nil
}

// FlushAll clears all data from Redis.
func (c *Client) FlushAll(ctx context.Context) error {
	if err := c.rdb.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush database: %w", err)
	}

	return nil
}
