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
