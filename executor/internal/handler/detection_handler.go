package handler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type DetectionHandler struct {
	// store in memory for now
	actions map[string]*models.ActionResult
	mu      sync.RWMutex
}

func NewDetectionHandler() *DetectionHandler {
	return &DetectionHandler{
		actions: map[string]*models.ActionResult{},
	}
}

func (h *DetectionHandler) HandleDetection(detection *models.Detection) (*models.ActionResult, error) {
	log.Printf("	Anomaly detected: [%s] - %s", detection.Severity, detection.Title)
	log.Printf("	Detector: %s", detection.DetectorName)
	log.Printf("	Action Type: %s", detection.ActionType)
	log.Printf("	Database: %s", detection.DatabaseID)
	log.Printf("	Recommendation: %s", detection.Recommendation) // TODO: Remove in future avoid verbosity in logs

	result := &models.ActionResult{
		ActionID:    generateActionID(),
		DetectionID: detection.DetectionID,
		ActionType:  detection.ActionType,
		DatabaseID:  detection.DatabaseID,
		Status:      models.StatusQueued,
		Message:     fmt.Sprintf("Action queued: %s", detection.ActionType),
		CreatedAt:   time.Now(),
		Completed:   nil,
	}

	// Keep in memory for now
	h.storeAction(result)

	log.Printf("Action queued: %s (ID: %s)", detection.ActionType, result.ActionID)

	return result, nil
}

func (h *DetectionHandler) GetActionStatus(actionID string) (*models.ActionResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	action, exists := h.actions[actionID]
	if !exists {
		return nil, fmt.Errorf("action not found: %s", actionID)
	}

	return action, nil
}

func (h *DetectionHandler) ListPendingActions(statusFilter string) ([]*models.ActionResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var results []*models.ActionResult

	for _, action := range h.actions {
		if statusFilter != "" && action.Status != statusFilter {
			continue
		}
		results = append(results, action)
	}

	log.Printf("Listed %d actions (filter: %s)", len(results), statusFilter)

	return results, nil
}

func (h *DetectionHandler) storeAction(action *models.ActionResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.actions[action.ActionID] = action
}

func generateActionID() string {
	return fmt.Sprintf("action-%d", time.Now().UnixNano())
}
