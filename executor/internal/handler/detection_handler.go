package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/joho/godotenv"
)

type DetectionHandler struct {
	// store in memory for now
	actions map[string]*models.ActionResult
	mu      sync.RWMutex

	dbConnection string
}

func NewDetectionHandler() *DetectionHandler {
	envPath := filepath.Join("..", ".env")
	_ = godotenv.Load(envPath)

	dbConnection := os.Getenv("DB_CONNECTION_STRING")
	if dbConnection == "" {
		log.Fatalf("No DB connection string in config")
	}
	return &DetectionHandler{
		actions:      map[string]*models.ActionResult{},
		dbConnection: dbConnection,
	}
}

func (h *DetectionHandler) HandleDetection(detection *models.Detection) (*models.ActionResult, error) {
	log.Printf("	Anomaly detected: [%s] - %s", detection.Severity, detection.Title)
	log.Printf("	Detector: %s", detection.DetectorName)
	log.Printf("	Action Type: %s", detection.ActionType)
	log.Printf("	Database: %s", detection.DatabaseID)

	actionID := generateActionID()

	action, err := h.createAction(detection, actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	go h.executeAction(action, detection)

	result := &models.ActionResult{
		ActionID:    actionID,
		DetectionID: detection.DetectionID,
		ActionType:  detection.ActionType,
		DatabaseID:  detection.DatabaseID,
		Status:      models.StatusQueued,
		Message:     fmt.Sprintf("Action queued: %s", detection.ActionType),
		CreatedAt:   time.Now(),
	}

	// Keep in memory for now
	h.storeAction(result)

	log.Printf("Action queued: %s (ID: %s)", detection.ActionType, result.ActionID)

	return result, nil
}

func (h *DetectionHandler) createAction(detection *models.Detection, actionID string) (actions.Action, error) {
	ctx := context.Background()
	// TODO: replace with factory in future
	adapter, err := database.NewPostgresAdapter(ctx, h.dbConnection, detection.DatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to create database adapter: %w", err)
	}

	metadata := &models.ActionMetadata{
		ActionID:     actionID,
		ActionType:   detection.ActionType,
		DatabaseID:   detection.DatabaseID,
		DatabaseType: "postgres", // TODO: get from detection
		CreatedAt:    time.Now(),
	}

	switch detection.ActionType {
	case "create_index":
		tableName, ok := detection.ActionMetaData["table_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing table_name in detection metadata")
		}

		columnName, ok := detection.ActionMetaData["column_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing column_name in detection metadata")
		}

		return actions.NewCreateIndexAction(metadata, adapter, tableName, []string{columnName}, false), nil

	// TODO: Add more actions here "deploy_pgbouncer" etc
	default:
		return nil, nil
	}
}

func (h *DetectionHandler) executeAction(action actions.Action, detection *models.Detection) {
	ctx := context.Background()
	metadata := action.GetMetadata()

	log.Printf("\tExecuting Action: %s (ID: %s)", metadata.ActionType, metadata.ActionID)

	result, err := action.Execute(ctx)
	if err != nil {
		log.Printf("Action execution failed: %v", err)
		result = &models.ActionResult{
			ActionID:    metadata.ActionID,
			DetectionID: detection.DetectionID,
			ActionType:  metadata.ActionType,
			DatabaseID:  metadata.DatabaseID,
			Status:      models.StatusFailed,
			Message:     "Execution error",
			Error:       err.Error(),
			CreatedAt:   metadata.CreatedAt,
		}
	}

	h.storeAction(result)

	if result.Status == models.StatusCompleted {
		log.Printf("\tAction Completed: %s (ID: %s)", metadata.ActionType, metadata.ActionID)
		log.Printf("\tChanges: %v", result.Changes)
	} else {
		log.Printf("\tAction Failed: %s (ID: %s)", metadata.ActionType, metadata.ActionID)
		log.Printf("\tError: %s", result.Error)
	}
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
