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
	Completed *time.Time

	// TODO: Execution details, rollback etc
}

const (
	StatusQueued    = "queued"
	StatusExecuting = "executing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)
