package handler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/eventbus"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

type DetectionHandler struct {
	actions         map[string]*models.ActionResult
	actionObjects   map[string]actions.Action
	mu              sync.RWMutex
	natsPublisher   *eventbus.Publisher
	knowledgeClient *knowledge.Client
}

func NewDetectionHandler(natsPublisher *eventbus.Publisher, knowledgeClient *knowledge.Client) *DetectionHandler {
	return &DetectionHandler{
		actions:         map[string]*models.ActionResult{},
		actionObjects:   map[string]actions.Action{},
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

	// Check execution mode
	executionMode := h.getExecutionMode(ctx)
	log.Printf("	Execution Mode: %s", executionMode)

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

	// Determine initial status based on execution mode
	var initialStatus string
	var message string

	switch executionMode {
	case models.ModeObserve:
		initialStatus = models.StatusSuggested
		message = fmt.Sprintf("Suggested action: %s (observe mode)", detection.ActionType)
	case models.ModeApproval:
		initialStatus = models.StatusPendingApproval
		message = fmt.Sprintf("Action pending approval: %s", detection.ActionType)
	default: // autonomous
		initialStatus = models.StatusQueued
		message = fmt.Sprintf("Action queued: %s", detection.ActionType)
	}

	result := &models.ActionResult{
		ActionID:    actionID,
		DetectionID: detection.DetectionID,
		ActionType:  detection.ActionType,
		DatabaseID:  detection.DatabaseID,
		Status:      initialStatus,
		Message:     message,
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

	log.Printf("Action %s: %s (ID: %s)", initialStatus, detection.ActionType, result.ActionID)

	// Only execute immediately in autonomous mode
	if executionMode == models.ModeAutonomous {
		go h.executeAction(action, detection)
	}

	return result, nil
}

func (h *DetectionHandler) getExecutionMode(ctx context.Context) string {
	if h.knowledgeClient == nil {
		return models.ModeAutonomous // Default if no Knowledge client
	}
	return h.knowledgeClient.GetExecutionMode(ctx)
}

// ApproveAction approves a pending action and executes it
func (h *DetectionHandler) ApproveAction(actionID string) (*models.ActionResult, error) {
	result, err := h.GetActionStatus(actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if result.Status != models.StatusPendingApproval {
		return nil, fmt.Errorf("action not pending approval, current status: %s", result.Status)
	}

	action, err := h.getActionObject(actionID)
	if err != nil {
		return nil, fmt.Errorf("action object not found: %w", err)
	}

	// Update status to approved/queued
	result.Status = models.StatusQueued
	result.Message = "Action approved by user"
	h.storeAction(result)

	ctx := context.Background()
	h.updateActionStatusInKnowledge(ctx, result)

	if h.natsPublisher != nil {
		h.natsPublisher.PublishActionStatus(result)
	}

	log.Printf("Action approved: %s", actionID)

	// Create detection from stored result for executeAction
	detection := &models.Detection{
		DetectionID: result.DetectionID,
		ActionType:  result.ActionType,
		DatabaseID:  result.DatabaseID,
	}

	// Execute the action
	go h.executeAction(action, detection)

	return result, nil
}

// RejectAction rejects a pending action
func (h *DetectionHandler) RejectAction(actionID string) (*models.ActionResult, error) {
	result, err := h.GetActionStatus(actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if result.Status != models.StatusPendingApproval {
		return nil, fmt.Errorf("action not pending approval, current status: %s", result.Status)
	}

	result.Status = models.StatusRejected
	result.Message = "Action rejected by user"
	h.storeAction(result)

	ctx := context.Background()
	h.updateActionStatusInKnowledge(ctx, result)

	if h.natsPublisher != nil {
		h.natsPublisher.PublishActionStatus(result)
	}

	log.Printf("Action rejected: %s", actionID)

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
		// Fetch database connection string from Knowledge
		if h.knowledgeClient == nil {
			return nil, fmt.Errorf("knowledge client not available - cannot fetch database connection")
		}

		dbResp, err := h.knowledgeClient.GetServiceClient().GetDatabase(ctx, &pb.GetDatabaseRequest{
			DatabaseId: detection.DatabaseID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch database connection from Knowledge: %w", err)
		}

		if !dbResp.Found {
			return nil, fmt.Errorf("database not found in Knowledge: %s", detection.DatabaseID)
		}

		dbConnectionString := dbResp.ConnectionString

		// TODO: replace with factory pattern when adding multi-database support
		adapter, err := database.NewPostgresAdapter(ctx, dbConnectionString, detection.DatabaseID)
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

	case "cache_optimization_recommendation":
		// Create recommendation action with safe and advanced options
		return actions.NewRecommendationAction(
			actionID,
			detection.DetectionID,
			detection.DatabaseID,
			metadata.DatabaseType, // Use from metadata
			detection.ActionMetaData,
		), nil

	// TODO: This is only implemented for PgBouncer, Analyser sends deploy_connection_pooler as a detection, make this choose based on DB later
	case "deploy_connection_pooler":
		action, err := actions.NewDeployPgBouncerAction(
			actionID,
			detection.DetectionID,
			detection.DatabaseID,
			"postgres",
			h.knowledgeClient.GetServiceClient(),
			detection.ActionMetaData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create PgBouncer action: %w", err)
		}
		return action, nil

	case "deploy_redis":
		// Deploy Redis cache layer (advanced - requires code changes)
		action, err := actions.NewDeployRedisAction(
			actionID,
			detection.DetectionID,
			detection.DatabaseID,
			detection.ActionMetaData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis action: %w", err)
		}
		return action, nil

	case "tune_config_high_latency":
		// Fetch database connection from Knowledge
		if h.knowledgeClient == nil {
			return nil, fmt.Errorf("knowledge client not available - cannot fetch database connection")
		}

		dbResp, err := h.knowledgeClient.GetServiceClient().GetDatabase(ctx, &pb.GetDatabaseRequest{
			DatabaseId: detection.DatabaseID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch database connection from Knowledge: %w", err)
		}

		if !dbResp.Found {
			return nil, fmt.Errorf("database not found in Knowledge: %s", detection.DatabaseID)
		}

		// Determine database type (default to postgres for now)
		databaseType := getStringFromMap(detection.ActionMetaData, "database_type", "postgres")

		// Create adapter based on database type
		var adapter database.DatabaseAdapter
		switch databaseType {
		case "postgres":
			adapter, err = database.NewPostgresAdapter(
				ctx,
				dbResp.ConnectionString,
				detection.DatabaseID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create postgres adapter: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported database type for config tuning: %s", databaseType)
		}

		// Create action with adapter
		return actions.NewTuneConfigAction(
			actionID,
			detection.DetectionID,
			detection.DatabaseID,
			databaseType,
			adapter,
		)

	case "optimise_queries":
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

// ExecuteActionDirectly executes an action without going through NATS detection flow.
// Used for user-triggered actions from Dashboard (e.g., manual Redis deployment).
func (h *DetectionHandler) ExecuteActionDirectly(action actions.Action, detection *models.Detection) {
	if action == nil {
		log.Printf("Warning: ExecuteActionDirectly called with nil action")
		return
	}

	metadata := action.GetMetadata()
	actionID := metadata.ActionID

	// Store action object for potential rollback
	h.storeActionObject(actionID, action)

	// Create initial result
	result := &models.ActionResult{
		ActionID:    actionID,
		DetectionID: detection.DetectionID,
		ActionType:  metadata.ActionType,
		DatabaseID:  metadata.DatabaseID,
		Status:      models.StatusQueued,
		Message:     fmt.Sprintf("Action queued: %s", metadata.ActionType),
		CreatedAt:   time.Now(),
	}
	h.storeAction(result)

	// Register with Knowledge if available
	if h.knowledgeClient != nil {
		if err := h.registerActionWithKnowledge(context.Background(), detection, result); err != nil {
			log.Printf("Warning: failed to register action with Knowledge: %v", err)
		}
	}

	// Publish queued status
	if h.natsPublisher != nil {
		h.natsPublisher.PublishActionStatus(result)
	}

	// Execute action (reuse existing executeAction method)
	h.executeAction(action, detection)
}

// Helper function to safely get string from map with default value
func getStringFromMap(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}
