package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/models"
)

func (c *Client) RegisterDetection(ctx context.Context, detection *models.Detection) error {
	detectionKey := fmt.Sprintf("detection:%s", detection.ID)

	data, err := json.Marshal(detection)
	if err != nil {
		return fmt.Errorf("failed to marshal detections: %w", err)
	}

	if err := c.rdb.Set(ctx, detectionKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store detections: %w", err)
	}

	keyMapping := fmt.Sprintf("detection_key:%s", detection.Key)
	if err := c.rdb.Set(ctx, keyMapping, detection.ID, 0).Err(); err != nil {
		return fmt.Errorf("failed to store key mapping: %w", err)
	}

	activeKey := fmt.Sprintf("detections:active:%s", detection.DatabaseID)
	if err := c.rdb.SAdd(ctx, activeKey, detection.ID).Err(); err != nil {
		return fmt.Errorf("failed to add active set: %w", err)
	}

	return nil
}

func (c *Client) IsDetectionActive(ctx context.Context, key string) (bool, error) {
	keyMapping := fmt.Sprintf("detection_key:%s", key)

	detectionID, err := c.rdb.Get(ctx, keyMapping).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil // Key doesnt exist
		}

		return false, fmt.Errorf("failed to check detection key: %w", err)
	}

	detection, err := c.GetDetection(ctx, detectionID)
	if err != nil {
		return false, err
	}

	return detection.State == models.StateActive, nil
}

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
		return fmt.Errorf("failed to remove from the active set: %w", err)
	}

	return nil
}

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

// ===== [ACTIONS OPERATIONS] =====

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
		return fmt.Errorf("failed to add to database set : %w", err)
	}

	statusKey := fmt.Sprintf("action:status:%s", action.Status)
	if err := c.rdb.SAdd(ctx, statusKey, action.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to status set: %w", err)
	}

	return nil
}

func (c *Client) UpdateActionStatus(ctx context.Context, actionID string, status models.ActionStatus, message string, errorMsg string) error {
	action, err := c.GetAction(ctx, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action for update: %w", err)
	}

	oldStatusKey := fmt.Sprintf("action:status:%s", action.Status)
	if err := c.rdb.SRem(ctx, oldStatusKey, actionID).Err(); err != nil {
		return fmt.Errorf("failed to remove old status set: %w", err)
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

func (c *Client) GetPendingActions(ctx context.Context, databaseID string) ([]*models.Action, error) {
	dbActionsKey := fmt.Sprintf("actions:database:%s", databaseID)

	actionIDs, err := c.rdb.SMembers(ctx, dbActionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get actions for %s : %w", databaseID, err)
	}

	actions := make([]*models.Action, 0)

	for _, id := range actionIDs {
		action, err := c.GetAction(ctx, id)
		if err != nil {
			continue // Skip errors
		}

		if action.Status == models.StatusQueued || action.Status == models.StatusExecuting {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

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
			continue // Skip errors
		}
		actions = append(actions, action)
	}

	return actions, nil
}
