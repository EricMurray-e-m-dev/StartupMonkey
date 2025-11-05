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
		sequentialScanThreshold: 1,
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

	//TODO: Remove after debugging
	fmt.Printf("DEBUG: snapshot.Labels: %+v\n", snapshot.Labels)
	fmt.Printf("DEBUG: snapshot.ExtendedMetrics: %+v\n", snapshot.ExtendedMetrics)

	worstTable, found := snapshot.Labels["pg.worst_seq_scan_table"]
	if !found || worstTable == "" {
		return d.createGenericDetection(snapshot, *seqScans)
	}

	prefix := fmt.Sprintf("pg.table.%s", worstTable)
	tableSeqScans := int64(snapshot.ExtendedMetrics[prefix+".seq_scans"])
	seqTupRead := int64(snapshot.ExtendedMetrics[prefix+".seq_tup_read"])

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = models.SeverityWarning
	detection.Timestamp = snapshot.Timestamp

	detection.Title = fmt.Sprintf("'Sequential Scans detected on table '%s''", worstTable)
	detection.Description = fmt.Sprintf(
		"Table '%s' is performing %d sequential scans (%d rows read)."+
			" Sequential scans read entire tables instead of using indexes, causes significant performance degredation under loads",
		worstTable, tableSeqScans, seqTupRead)

	detection.Evidence = map[string]interface{}{
		"table_name":       worstTable,
		"sequential_scans": seqScans,
		"rows_read":        seqTupRead,
		"query_health":     snapshot.QueryHealth,
	}

	detection.Recommendation = fmt.Sprintf(
		"Table '%s' needs an index. Run EXPLAIN ANALYZE on queries using this table "+
			"to identify which columns are frequently filtered or joined. "+
			"Create indexes on columns used in WHERE clauses, JOIN conditions, and ORDER BY statements. "+
			"Use CREATE INDEX CONCURRENTLY to avoid blocking production queries.",
		worstTable,
	)

	detection.ActionType = "create_index"
	detection.ActionMetadata = map[string]interface{}{
		"table_name": worstTable,
		"priority":   "medium",
	}

	return detection
}

func (d *MissingIndexDetector) SetThreshold(threshold int32) {
	d.sequentialScanThreshold = threshold
}

func (d *MissingIndexDetector) createGenericDetection(snapshot *normaliser.NormalisedMetrics, seqScans int32) *models.Detection {
	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = models.SeverityWarning
	detection.Timestamp = snapshot.Timestamp

	detection.Title = "Sequential Scans detected"
	detection.Description = fmt.Sprintf(
		"Database is performing %d sequential scans, indicating missing indexes on frequently queried tables. "+
			"Sequential scans read entire tables instead of using indexes, causing significant performance degradation under load.",
		seqScans,
	)

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
