package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type LongRunningQueryDetector struct {
	thresholdSecs float64
}

func NewLongRunningQueryDetector() *LongRunningQueryDetector {
	return &LongRunningQueryDetector{
		thresholdSecs: 30.0, // 30 seconds default
	}
}

func (d *LongRunningQueryDetector) Name() string {
	return "long_running_query"
}

func (d *LongRunningQueryDetector) Category() models.DetectionCategory {
	return models.CategoryQuery
}

func (d *LongRunningQueryDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	duration, found := snapshot.ExtendedMetrics["pg.longest_query_duration_secs"]
	if !found || duration < d.thresholdSecs {
		return nil
	}

	pid, hasPID := snapshot.Labels["pg.longest_query_pid"]
	if !hasPID {
		return nil
	}

	username := snapshot.Labels["pg.longest_query_user"]
	queryText := snapshot.Labels["pg.longest_query_text"]

	var severity models.DetectionSeverity
	if duration >= 120 {
		severity = models.SeverityCritical
	} else if duration >= 60 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	detection.Title = fmt.Sprintf("Long-running query detected (%.0fs)", duration)
	detection.Description = fmt.Sprintf(
		"Query running for %.0f seconds by user '%s'. "+
			"Long-running queries can hold locks, consume resources, and block other operations. "+
			"Query: %s",
		duration, username, queryText,
	)

	detection.Evidence = map[string]interface{}{
		"pid":           pid,
		"username":      username,
		"query":         queryText,
		"duration_secs": duration,
	}

	detection.Recommendation = fmt.Sprintf(
		"Consider terminating the query (PID %s) if it's not critical. "+
			"Investigate why the query is running slowly - it may need index optimisation "+
			"or query restructuring.",
		pid,
	)

	detection.ActionType = "terminate_query"
	detection.ActionMetadata = map[string]interface{}{
		"pid":      pid,
		"username": username,
		"graceful": true, // pg_cancel_backend first, pg_terminate_backend if needed
	}

	return detection
}

func (d *LongRunningQueryDetector) SetThreshold(thresholdSecs float64) {
	d.thresholdSecs = thresholdSecs
}
