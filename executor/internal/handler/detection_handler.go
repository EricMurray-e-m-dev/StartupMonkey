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
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/eventbus"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"github.com/joho/godotenv"
)

type DetectionHandler struct {
	actions         map[string]*models.ActionResult
	actionObjects   map[string]actions.Action
	mu              sync.RWMutex
	dbConnection    string
	natsPublisher   *eventbus.Publisher
	knowledgeClient *knowledge.Client
}

func NewDetectionHandler(natsPublisher *eventbus.Publisher, knowledgeClient *knowledge.Client) *DetectionHandler {
	envPath := filepath.Join("..", ".env")
	_ = godotenv.Load(envPath)

	dbConnection := os.Getenv("DB_CONNECTION_STRING")
	if dbConnection == "" {
		log.Fatalf("No DB connection string in config")
	}
	return &DetectionHandler{
		actions:         map[string]*models.ActionResult{},
		actionObjects:   map[string]actions.Action{},
		dbConnection:    dbConnection,
		natsPublisher:   natsPublisher,
		knowledgeClient: knowledgeClient,
	}
}

func (h *DetectionHandler) HandleDetection(detection *models.Detection) (*models.ActionResult, error) {
	log.Printf("	Anomaly detected: [%s] - %s", detection.Severity, detection.Title)
	log.Printf("	Detector: %s", detection.DetectorName)
	log.Printf("	Action Type: %s", detection.ActionType)
	log.Printf("	Database: %s", detection.DatabaseID)

	ctx := context.Background()

	if h.knowledgeClient != nil {
		if isDuplicate, err := h.checkForDuplicateActions(ctx, detection); err != nil {
			log.Printf("warning failed to check duplicate actions: %v", err)
		} else if isDuplicate {
			log.Printf("Action already pending for detection, skipping")
			return nil, nil
		}
	}

	actionID := generateActionID()

	action, err := h.createAction(detection, actionID)
	if err != nil {
		log.Printf("failed to create action: %v", err)
		return nil, err
	}

	h.storeActionObject(actionID, action)

	result := &models.ActionResult{
		ActionID:    actionID,
		DetectionID: detection.DetectionID,
		ActionType:  detection.ActionType,
		DatabaseID:  detection.DatabaseID,
		Status:      models.StatusQueued,
		Message:     fmt.Sprintf("Action queued: %s", detection.ActionType),
		CreatedAt:   time.Now(),
	}

	if h.knowledgeClient != nil {
		if err := h.registerActionWithKnowledge(ctx, detection, result); err != nil {
			log.Printf("warning failed to register action with knowledge: %v", err)
		} else {
			log.Printf("Action registered with knowledge")
		}
	}

	h.storeAction(result)

	if h.natsPublisher != nil {
		if err := h.natsPublisher.PublishActionStatus(result); err != nil {
			log.Printf("Warning: failed to publish action status to event bus: %v", err)
		}
	}

	log.Printf("Action queued: %s (ID: %s)", detection.ActionType, result.ActionID)

	go h.executeAction(action, detection)

	return result, nil
}

func (h *DetectionHandler) createAction(detection *models.Detection, actionID string) (actions.Action, error) {
	ctx := context.Background()

	metadata := &models.ActionMetadata{
		ActionID:     actionID,
		ActionType:   detection.ActionType,
		DatabaseID:   detection.DatabaseID,
		DatabaseType: "postgres", // TODO: get from detection
		CreatedAt:    time.Now(),
	}

	switch detection.ActionType {
	case "create_index":
		// TODO: replace with factory pattern when adding multi-database support
		adapter, err := database.NewPostgresAdapter(ctx, h.dbConnection, detection.DatabaseID)
		if err != nil {
			return nil, fmt.Errorf("failed to create database adapter: %w", err)
		}

		tableName, ok := detection.ActionMetaData["table_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing table_name in detection metadata")
		}

		columnName, ok := detection.ActionMetaData["column_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing column_name in detection metadata")
		}

		return actions.NewCreateIndexAction(metadata, adapter, tableName, []string{columnName}, false), nil

	case "deploy_pgbouncer":
		action, err := actions.NewDeployPgBouncerAction(
			actionID,
			detection.DetectionID,
			detection.DatabaseID,
			"postgres",
			detection.ActionMetaData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create PgBouncer action: %w", err)
		}
		return action, nil

	case "increase_cache_size", "deploy_redis", "optimise_queries":
		return actions.NewFutureFixAction(
			actionID,
			detection.ActionType,
			detection.DatabaseID,
			fmt.Sprintf("Action '%s' is not yet implemented. This action has been queued for future implementation.", detection.ActionType),
		), nil

	default:
		return nil, fmt.Errorf("action type not implemented yet: %s", detection.ActionType)
	}
}

func (h *DetectionHandler) executeAction(action actions.Action, detection *models.Detection) {
	if action == nil {
		log.Printf("Warning: executeAction called with nil action for detection %s", detection.DetectionID)
		return
	}

	ctx := context.Background()
	metadata := action.GetMetadata()

	log.Printf("\tExecuting Action: %s (ID: %s)", metadata.ActionType, metadata.ActionID)

	executingResult := &models.ActionResult{
		ActionID:    metadata.ActionID,
		DetectionID: detection.DetectionID,
		ActionType:  metadata.ActionType,
		DatabaseID:  metadata.DatabaseID,
		Status:      models.StatusExecuting,
		Message:     "Action executing",
		CreatedAt:   metadata.CreatedAt,
	}
	h.storeAction(executingResult)

	if h.natsPublisher != nil {
		h.natsPublisher.PublishActionStatus(executingResult)
	}

	h.updateActionStatusInKnowledge(ctx, executingResult)

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

	h.updateActionStatusInKnowledge(ctx, result)

	if h.natsPublisher != nil {
		if err := h.natsPublisher.PublishActionStatus(result); err != nil {
			log.Printf("Warning: failed to publish action status to event bus: %v", err)
		}

		if result.Status == models.StatusCompleted {
			if err := h.natsPublisher.PublishActionCompleted(result, detection); err != nil {
				log.Printf("Warning: failed to publish action completion: %v", err)
			}
		}
	}

	switch result.Status {
	case models.StatusCompleted:
		log.Printf("\tAction Completed: %s (ID: %s)", metadata.ActionType, metadata.ActionID)
		log.Printf("\tChanges: %v", result.Changes)
	case models.StatusPendingImplementation:
		log.Printf("\t⏸Action Pending Implementation: %s (ID: %s)", metadata.ActionType, metadata.ActionID)
		log.Printf("\tReason: %s", result.Message)
	default:
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

func (h *DetectionHandler) RollbackAction(actionID string) (*models.ActionResult, error) {
	result, err := h.GetActionStatus(actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if !result.CanRollback {
		return nil, fmt.Errorf("action does not support rollback")
	}

	if result.Status != models.StatusCompleted {
		return nil, fmt.Errorf("can only rollback completed actions, current status: %s", result.Status)
	}

	action, err := h.getActionObject(actionID)
	if err != nil {
		return nil, fmt.Errorf("action object not found: %w", err)
	}

	ctx := context.Background()
	err = action.Rollback(ctx)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	result.Status = models.StatusRolledBack
	result.Rolledback = true
	result.Message = "Action rolled back successfully"
	h.storeAction(result)

	if h.knowledgeClient != nil {
		h.updateActionStatusInKnowledge(ctx, result)
	}

	if h.natsPublisher != nil {
		h.natsPublisher.PublishActionStatus(result)
	}

	log.Printf("Action rolled back: %s", actionID)

	return result, nil
}

func (h *DetectionHandler) storeAction(action *models.ActionResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.actions[action.ActionID] = action
}

func generateActionID() string {
	return fmt.Sprintf("action-%d", time.Now().UnixNano())
}

func (h *DetectionHandler) checkForDuplicateActions(ctx context.Context, detection *models.Detection) (bool, error) {
	pendingActions, err := h.knowledgeClient.GetPendingActions(ctx, detection.DatabaseID)
	if err != nil {
		return false, err
	}

	for _, pending := range pendingActions {
		if pending.DetectionId == detection.DetectionID {
			return true, nil
		}
	}

	return false, nil
}

func (h *DetectionHandler) registerActionWithKnowledge(ctx context.Context, detection *models.Detection, result *models.ActionResult) error {
	return h.knowledgeClient.RegisterAction(ctx, &pb.RegisterActionRequest{
		Id:          result.ActionID,
		DetectionId: detection.DetectionID,
		ActionType:  result.ActionType,
		DatabaseId:  result.DatabaseID,
		CreatedAt:   result.CreatedAt.Unix(),
	})
}

func (h *DetectionHandler) updateActionStatusInKnowledge(ctx context.Context, result *models.ActionResult) {
	if h.knowledgeClient == nil {
		return
	}

	err := h.knowledgeClient.UpdateActionStatus(ctx, &pb.UpdateActionRequest{
		ActionId:  result.ActionID,
		Status:    string(result.Status),
		Message:   result.Message,
		Error:     result.Error,
		Timestamp: time.Now().Unix(),
	})

	if err != nil {
		log.Printf("warning failed to update action in knowledge: %v", err)
	} else {
		log.Printf("Action updated in knowledge: %s -> %s", result.ActionID, result.Status)
	}
}

func (h *DetectionHandler) storeActionObject(actionID string, action actions.Action) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actionObjects[actionID] = action
}

func (h *DetectionHandler) getActionObject(actionID string) (actions.Action, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	action, exists := h.actionObjects[actionID]
	if !exists {
		return nil, fmt.Errorf("action object does not exists: %s", actionID)
	}

	return action, nil
}
