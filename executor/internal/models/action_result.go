package models

import "time"

type ActionResult struct {
	ActionID    string
	DetectionID string
	ActionType  string
	DatabaseID  string

	Status    string
	Message   string
	CreatedAt time.Time
	Started   *time.Time
	Completed *time.Time

	ExecutionTimeMs int64
	Changes         map[string]interface{}
	Error           string

	CanRollback   bool
	Rolledback    bool
	RollbackError string
}

type ActionMetadata struct {
	ActionID     string
	ActionType   string
	DatabaseID   string
	DatabaseType string
	CreatedAt    time.Time
}

const (
	StatusQueued    = "queued"
	StatusExecuting = "executing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)
