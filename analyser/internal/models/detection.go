package models

import "time"

// DetectionCategories for grouping similar issues
type DetectionCategory string

const (
	CategoryQuery      DetectionCategory = "query"
	CategoryConnection DetectionCategory = "connection"
	CategoryCache      DetectionCategory = "cache"
	CategoryStorage    DetectionCategory = "storage"
)

// DetectionSeverity indicates urgency
type DetectionSeverity string

const (
	SeverityInfo     DetectionSeverity = "info"
	SeverityWarning  DetectionSeverity = "warning"
	SeverityCritical DetectionSeverity = "critical"
)

// Detection holds info on a detected issue
type Detection struct {
	ID           string            `json:"id"`
	DetectorName string            `json:"detector_name"` // Detector that found the issue
	Category     DetectionCategory `json:"category"`
	Severity     DetectionSeverity `json:"severity"`

	DatabaseID string `json:"database_id"`
	Timestamp  int64  `json:"timestamp"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Evidence map[string]interface{} `json:"evidence"`

	Recommendation string `json:"recommendation"`

	// For executor in future
	ActionType     string                 `json:"action_type,omitempty"`
	ActionMetadata map[string]interface{} `json:"action_metadata,omitempty"`
}

func NewDetection(detectorName string, category DetectionCategory, databaseId string) *Detection {
	return &Detection{
		ID:             generateDetectionID(detectorName, time.Now().Unix()),
		DetectorName:   detectorName,
		Category:       category,
		DatabaseID:     databaseId,
		Timestamp:      time.Now().Unix(),
		Evidence:       make(map[string]interface{}),
		ActionMetadata: make(map[string]interface{}),
	}
}

func generateDetectionID(detectorName string, timestamp int64) string {
	return detectorName + "-" + string(rune(timestamp))
}
