package models

import "time"

type ActionStatus string

const (
	StatusQueued    ActionStatus = "queued"
	StatusExecuting ActionStatus = "executing"
	StatusCompleted ActionStatus = "completed"
	StatusFailed    ActionStatus = "failed"
)

type Action struct {
	ID          string       `json:"id"`
	DetectionID string       `json:"detection_id"`
	ActionType  string       `json:"action_type"`
	DatabaseID  string       `json:"database_id"`
	Status      ActionStatus `json:"status"`
	Message     string       `json:"message"`
	Error       string       `json:"error,omitempty"`
	Result      string       `json:"result,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}
