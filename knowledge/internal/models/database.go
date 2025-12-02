package models

import "time"

type Database struct {
	ID               string            `json:"id"`
	ConnectionString string            `json:"connection_string"`
	DatabaseType     string            `json:"database_type"`
	DatabaseName     string            `json:"database_name"`
	Host             string            `json:"host"`
	Port             int32             `json:"port"`
	Version          string            `json:"version"`
	RegisteredAt     time.Time         `json:"registered_at"`
	LastSeen         time.Time         `json:"last_seen"`
	Status           string            `json:"status"`
	HealthScore      float64           `json:"health_score"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}
