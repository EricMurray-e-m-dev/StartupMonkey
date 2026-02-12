package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type IdleTransactionDetector struct {
	thresholdSecs float64
}

func NewIdleTransactionDetector() *IdleTransactionDetector {
	return &IdleTransactionDetector{
		thresholdSecs: 300.0, // 5 minutes default
	}
}

func (d *IdleTransactionDetector) Name() string {
	return "idle_transaction"
}

func (d *IdleTransactionDetector) Category() models.DetectionCategory {
	return models.CategoryConnection
}

func (d *IdleTransactionDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	duration, found := snapshot.ExtendedMetrics["pg.idle_txn_duration_secs"]
	if !found || duration < d.thresholdSecs {
		return nil
	}

	pid, hasPID := snapshot.Labels["pg.idle_txn_pid"]
	if !hasPID {
		return nil
	}

	username := snapshot.Labels["pg.idle_txn_user"]
	query := snapshot.Labels["pg.idle_txn_query"]

	var severity models.DetectionSeverity
	if duration >= 900 { // 15 minutes
		severity = models.SeverityCritical
	} else if duration >= 600 { // 10 minutes
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	durationMins := duration / 60
	detection.Title = fmt.Sprintf("Idle transaction detected (%.0f minutes)", durationMins)
	detection.Description = fmt.Sprintf(
		"Connection held by user '%s' has been idle in transaction for %.0f minutes. "+
			"Idle transactions hold locks, block VACUUM, and consume connection slots. "+
			"Last query: %s",
		username, durationMins, query,
	)

	detection.Evidence = map[string]interface{}{
		"pid":                pid,
		"username":           username,
		"query":              query,
		"idle_duration_secs": duration,
		"idle_duration_mins": durationMins,
	}

	detection.Recommendation = fmt.Sprintf(
		"Terminate the idle connection (PID %s) to release locks and free the connection slot. "+
			"Investigate the application code to ensure transactions are properly committed or rolled back.",
		pid,
	)

	detection.ActionType = "terminate_query"
	detection.ActionMetadata = map[string]interface{}{
		"pid":      pid,
		"username": username,
		"graceful": false, // Idle transactions should be terminated, not cancelled
	}

	return detection
}

func (d *IdleTransactionDetector) SetThreshold(thresholdSecs float64) {
	d.thresholdSecs = thresholdSecs
}
