package detector

import (
	"fmt"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

type TableBloatDetector struct {
	bloatRatioThreshold float64
}

func NewTableBloatDetector() *TableBloatDetector {
	return &TableBloatDetector{
		bloatRatioThreshold: 0.1, // 10% dead tuples
	}
}

func (d *TableBloatDetector) Name() string {
	return "table_bloat"
}

func (d *TableBloatDetector) Category() models.DetectionCategory {
	return models.CategoryStorage
}

func (d *TableBloatDetector) Detect(snapshot *normaliser.NormalisedMetrics) *models.Detection {
	worstTable, found := snapshot.Labels["pg.worst_bloat_table"]
	if !found || worstTable == "" {
		return nil
	}

	bloatRatio, found := snapshot.ExtendedMetrics["pg.worst_bloat_ratio"]
	if !found || bloatRatio < d.bloatRatioThreshold {
		return nil
	}

	prefix := fmt.Sprintf("pg.table.%s", worstTable)
	liveTuples := int64(snapshot.ExtendedMetrics[prefix+".live_tuples"])
	deadTuples := int64(snapshot.ExtendedMetrics[prefix+".dead_tuples"])

	var severity models.DetectionSeverity
	if bloatRatio >= 0.3 {
		severity = models.SeverityCritical
	} else if bloatRatio >= 0.2 {
		severity = models.SeverityWarning
	} else {
		severity = models.SeverityInfo
	}

	detection := models.NewDetection(d.Name(), d.Category(), snapshot.DatabaseID)
	detection.Severity = severity
	detection.Timestamp = snapshot.Timestamp

	bloatPercent := int(bloatRatio * 100)
	detection.Title = fmt.Sprintf("Table bloat detected on '%s' (%d%% dead tuples)", worstTable, bloatPercent)
	detection.Description = fmt.Sprintf(
		"Table '%s' has %d dead tuples out of %d live tuples (%.1f%% bloat). "+
			"Dead tuples consume disk space and slow down queries. "+
			"Running VACUUM will reclaim space and improve performance.",
		worstTable, deadTuples, liveTuples, bloatRatio*100,
	)

	detection.Evidence = map[string]interface{}{
		"table_name":    worstTable,
		"live_tuples":   liveTuples,
		"dead_tuples":   deadTuples,
		"bloat_ratio":   bloatRatio,
		"bloat_percent": bloatPercent,
	}

	detection.Recommendation = fmt.Sprintf(
		"Run VACUUM ANALYZE on table '%s' to reclaim space from dead tuples "+
			"and update query planner statistics. This operation is non-blocking "+
			"and safe to run on production databases.",
		worstTable,
	)

	detection.ActionType = "vacuum_table"
	detection.ActionMetadata = map[string]interface{}{
		"table_name": worstTable,
		"priority":   d.getPriority(bloatRatio),
	}

	return detection
}

func (d *TableBloatDetector) getPriority(bloatRatio float64) string {
	if bloatRatio >= 0.3 {
		return "high"
	} else if bloatRatio >= 0.2 {
		return "medium"
	}
	return "low"
}

func (d *TableBloatDetector) SetThreshold(threshold float64) {
	d.bloatRatioThreshold = threshold
}
