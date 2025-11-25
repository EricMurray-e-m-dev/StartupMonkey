package detector

import (
	"fmt"

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

	worstTable, found := snapshot.Labels["pg.worst_seq_scan_table"]
	if !found || worstTable == "" {
		// TODO: Implement Fallback
		return nil
	}

	recommendedColumn, hasColumn := snapshot.Labels["pg.recommended_index_column"]
	if !hasColumn || recommendedColumn == "" {
		// TODO: Implement Fallback
		return nil
	}

	prefix := fmt.Sprintf("pg.table.%s", worstTable)
	tableSeqScans := int64(snapshot.ExtendedMetrics[prefix+".seq_scans"])
	seqTupRead := int64(snapshot.ExtendedMetrics[prefix+".seq_tup_read"])

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
			"This column was identified through query analysis. "+
			"Use CREATE INDEX CONCURRENTLY to avoid blocking production queries.",
		worstTable, recommendedColumn,
	)

	detection.ActionType = "create_index"
	detection.ActionMetadata = map[string]interface{}{
		"table_name":  worstTable,
		"column_name": recommendedColumn,
		"priority":    "high",
	}

	return detection
}

func (d *MissingIndexDetector) SetThreshold(threshold int32) {
	d.sequentialScanThreshold = threshold
}

func (d *MissingIndexDetector) SetDeltaThreshold(threshold float64) {
	d.sequentialScanDeltaThreshold = threshold
}
