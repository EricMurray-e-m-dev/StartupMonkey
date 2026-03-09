package detector

import (
	"fmt"
	"strings"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type MissingIndexDetector struct {
	sequentialScanThreshold      int32
	sequentialScanDeltaThreshold float64
}

func NewMissingIndexDetector() *MissingIndexDetector {
	return &MissingIndexDetector{
		sequentialScanThreshold:      1,
		sequentialScanDeltaThreshold: 10.0,
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

	if snapshot.MetricDeltas != nil {
		if delta, exists := snapshot.MetricDeltas["sequential_scans"]; exists {
			if delta <= d.sequentialScanDeltaThreshold {
				return nil
			}
		}
	} else {
		if *seqScans <= d.sequentialScanThreshold {
			return nil
		}
	}

	// Database-agnostic label lookup
	prefix, worstTable := findLabelBySuffix(snapshot.Labels, "worst_seq_scan_table")
	if worstTable == "" {
		return nil
	}

	_, recommendedColumn := findLabelBySuffix(snapshot.Labels, "recommended_index_column")
	if recommendedColumn == "" {
		return nil
	}

	// Use the same prefix for extended metrics (e.g., "pg.table." or "mysql.table.")
	tablePrefix := fmt.Sprintf("%s.table.%s", prefix, worstTable)
	tableSeqScans := int64(snapshot.ExtendedMetrics[tablePrefix+".seq_scans"])
	seqTupRead := int64(snapshot.ExtendedMetrics[tablePrefix+".seq_tup_read"])

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = models.SeverityWarning
	detection.Timestamp = snapshot.Timestamp

	detection.Title = fmt.Sprintf("Sequential scans detected on table '%s'", worstTable)
	detection.Description = fmt.Sprintf(
		"Table '%s' is performing %d sequential scans (%d rows read). "+
			"Column '%s' is frequently filtered in queries without an index, "+
			"causing full table scans.",
		worstTable, tableSeqScans, seqTupRead, recommendedColumn,
	)

	detection.Evidence = map[string]interface{}{
		"table_name":       worstTable,
		"column_name":      recommendedColumn,
		"sequential_scans": tableSeqScans,
		"rows_read":        seqTupRead,
		"query_health":     snapshot.QueryHealth,
		"database_type":    snapshot.DatabaseType,
	}

	if snapshot.MetricDeltas != nil {
		if delta, exists := snapshot.MetricDeltas["sequential_scans"]; exists {
			detection.Evidence["sequential_scans_delta"] = delta
		}
		if snapshot.TimeDeltaSeconds > 0 {
			detection.Evidence["time_delta_seconds"] = snapshot.TimeDeltaSeconds
		}
	}

	detection.Recommendation = fmt.Sprintf(
		"Create an index on %s.%s to optimize query performance. "+
			"This column was identified through query analysis.",
		worstTable, recommendedColumn,
	)

	detection.ActionType = "create_index"
	detection.ActionMetadata = map[string]interface{}{
		"table_name":    worstTable,
		"column_name":   recommendedColumn,
		"database_type": snapshot.DatabaseType,
		"priority":      "high",
	}

	return detection
}

// findLabelBySuffix searches for a label ending with the given suffix.
// Returns the prefix (e.g., "pg", "mysql") and the value.
func findLabelBySuffix(labels map[string]string, suffix string) (string, string) {
	for key, value := range labels {
		if strings.HasSuffix(key, "."+suffix) {
			// Extract prefix (everything before the last dot + suffix)
			prefix := strings.TrimSuffix(key, "."+suffix)
			return prefix, value
		}
	}
	return "", ""
}

func (d *MissingIndexDetector) SetThreshold(threshold int32) {
	d.sequentialScanThreshold = threshold
}

func (d *MissingIndexDetector) SetDeltaThreshold(threshold float64) {
	d.sequentialScanDeltaThreshold = threshold
}
