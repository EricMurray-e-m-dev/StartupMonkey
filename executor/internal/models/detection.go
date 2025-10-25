package models

import "time"

type Detection struct {
	DetectionID  string
	DetectorName string
	Category     string
	Severity     string
	DatabaseID   string

	Title          string
	Description    string
	Recommendation string

	ActionType     string
	ActionMetaData map[string]interface{}
	Evidence       map[string]interface{}

	Timestamp time.Time
}
