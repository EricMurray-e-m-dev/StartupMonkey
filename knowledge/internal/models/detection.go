package models

import "time"

type DetectionState string

const (
	StateActive     DetectionState = "active"
	StateResolved   DetectionState = "resolved"
	StateSuperseded DetectionState = "superseded"
)

type Detection struct {
	ID         string         `json:"id"`
	Key        string         `json:"key"`
	State      DetectionState `json:"state"`
	Severity   string         `json:"severity"`
	Category   string         `json:"category"`
	DatabaseID string         `json:"database_id"`
	Value      float64        `json:"value"`
	ActionID   string         `json:"action_id"`
	ResolvedBy string         `json:"resolved_by"`
	CreatedAt  time.Time      `json:"created_at"`
	LastSeen   time.Time      `json:"last_seen"`
	TTL        int            `json:"ttl"`
}
