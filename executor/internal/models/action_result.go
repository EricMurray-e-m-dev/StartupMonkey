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
	StatusQueued    = "queued"
	StatusExecuting = "executing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)
