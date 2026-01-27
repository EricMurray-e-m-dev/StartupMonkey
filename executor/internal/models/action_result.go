package models

import "time"

type ActionResult struct {
	ActionID    string `json:"action_id"`
	DetectionID string `json:"detection_id"`
	ActionType  string `json:"action_type"`
	DatabaseID  string `json:"database_id"`

	Status    string     `json:"status"`
	Message   string     `json:"message"`
	CreatedAt time.Time  `json:"created_at"`
	Started   *time.Time `json:"started,omitempty"`
	Completed *time.Time `json:"completed,omitempty"`

	ExecutionTimeMs int64                  `json:"execution_time_ms"`
	Changes         map[string]interface{} `json:"changes,omitempty"`
	Error           string                 `json:"error,omitempty"`

	CanRollback   bool   `json:"can_rollback"`
	Rolledback    bool   `json:"rolledback"`
	RollbackError string `json:"rollback_error,omitempty"`
}

type ActionMetadata struct {
	ActionID     string    `json:"action_id"`
	ActionType   string    `json:"action_type"`
	DatabaseID   string    `json:"database_id"`
	DatabaseType string    `json:"database_type"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	StatusQueued                = "queued"
	StatusSuggested             = "suggested"        // Observe mode - recommendation only
	StatusPendingApproval       = "pending_approval" // Approval mode - waiting for user
	StatusApproved              = "approved"         // User approved, ready to execute
	StatusRejected              = "rejected"         // User rejected
	StatusExecuting             = "executing"
	StatusCompleted             = "completed"
	StatusFailed                = "failed"
	StatusPendingImplementation = "pending_implementation"
	StatusRolledBack            = "rolled_back"
)

// Execution modes
const (
	ModeObserve    = "observe"    // Detect only, no execution
	ModeApproval   = "approval"   // Detect, wait for approval
	ModeAutonomous = "autonomous" // Detect and execute immediately
)
