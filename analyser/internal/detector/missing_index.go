package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type MissingIndexDetector struct {
	sequentialScanThreshold int32
}

func NewMissingIndexDetector() *MissingIndexDetector {
	return &MissingIndexDetector{
		sequentialScanThreshold: 10, // Alert if over 10 seq scans
	}
}

func (d *MissingIndexDetector) Name() string {
	return "missing_index"
}

func (d *MissingIndexDetector) Category() models.DetectionCategory {
	return models.CategoryQuery
}

func (d *MissingIndexDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	if snapshot.Measurements.SequentialScans == nil {
		return nil
	}

	seqScans := snapshot.Measurements.SequentialScans

	if *seqScans <= d.sequentialScanThreshold {
		return nil
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = models.SeverityWarning
	detection.Timestamp = snapshot.Timestamp

	detection.Title = "Sequential Scans detected"
	detection.Description = fmt.Sprintf(
		"Database is performing %d sequential scans, indicating missing indexs on frequently queried tables."+
			" Sequential scans read entire tables instead of using indexes, causes significant performance degredation under loads",
		*seqScans)

	detection.Evidence = map[string]interface{}{
		"sequential_scans": seqScans,
		"threshold":        d.sequentialScanThreshold,
		"query_health":     snapshot.QueryHealth,
	}

	detection.Recommendation = "Run EXPLAIN ANALYZE on your slowest queries to identify which tables are being scanned sequentially. " +
		"Create indexes on columns frequently used in WHERE clauses, JOIN conditions, and ORDER BY statements. " +
		"Use CREATE INDEX CONCURRENTLY to avoid blocking production queries."

	detection.ActionType = "create_index"
	detection.ActionMetadata = map[string]interface{}{
		"analysis_required": true,
		"priority":          "medium",
	}

	return detection
}

func (d *MissingIndexDetector) SetThreshold(threshold int32) {
	d.sequentialScanThreshold = threshold
}
