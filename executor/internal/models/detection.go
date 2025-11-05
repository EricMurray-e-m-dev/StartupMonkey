package models

type Detection struct {
	DetectionID    string                 `json:"id"` // Match Analyser's "id"
	DetectorName   string                 `json:"detector_name"`
	Category       string                 `json:"category"`
	Severity       string                 `json:"severity"`
	DatabaseID     string                 `json:"database_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Recommendation string                 `json:"recommendation"`
	ActionType     string                 `json:"action_type"`
	ActionMetaData map[string]interface{} `json:"action_metadata"` // Match Analyser's "action_metadata"
	Evidence       map[string]interface{} `json:"evidence"`
	Timestamp      int64                  `json:"timestamp"`
}
