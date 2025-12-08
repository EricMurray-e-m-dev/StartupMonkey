package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/database"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type TuneConfigAction struct {
	actionID       string
	detectionID    string
	databaseID     string
	databaseType   string
	adapter        database.DatabaseAdapter
	originalConfig map[string]string
	appliedChanges map[string]string
}

// NewTuneConfigAction creates a new config tuning action with injected adapter
func NewTuneConfigAction(
	actionID string,
	detectionID string,
	databaseID string,
	databaseType string,
	adapter database.DatabaseAdapter,
) (Action, error) {
	// Validate adapter supports config tuning
	caps := adapter.GetCapabilities()
	if !caps.SupportsConfigTuning {
		return nil, fmt.Errorf("database %s does not support config tuning", databaseType)
	}

	return &TuneConfigAction{
		actionID:     actionID,
		detectionID:  detectionID,
		databaseID:   databaseID,
		databaseType: databaseType,
		adapter:      adapter,
	}, nil
}

func (a *TuneConfigAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	log.Printf("Tuning configuration for database: %s", a.databaseID)

	// Parameters to tune (PostgreSQL-specific for now)
	parameters := []string{"work_mem", "effective_cache_size", "random_page_cost"}

	// 1. Get current configuration
	currentConfig, err := a.adapter.GetCurrentConfig(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to get current config: %w", err)
	}

	log.Printf("Current config: %+v", currentConfig)
	a.originalConfig = currentConfig

	// 2. Determine optimal configuration
	newConfig := a.calculateOptimalConfig(currentConfig)

	// 3. Apply configuration changes
	err = a.adapter.SetConfig(ctx, newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to apply config changes: %w", err)
	}

	log.Printf("Applied config changes: %+v", newConfig)
	a.appliedChanges = newConfig

	// 4. Get slow queries for educational component
	slowQueries, err := a.adapter.GetSlowQueries(ctx, 500.0, 5) // 500ms threshold, top 5
	if err != nil {
		log.Printf("Warning: failed to retrieve slow queries: %v", err)
		slowQueries = []database.SlowQuery{} // Continue without slow queries
	}

	// 5. Build optimization guide
	guide := a.getOptimizationGuide()

	// 6. Build result
	changes := map[string]interface{}{
		"config_changes":     newConfig,
		"original_config":    currentConfig,
		"slow_queries":       slowQueries,
		"optimization_guide": guide,
		"database_type":      a.databaseType,
	}

	return &models.ActionResult{
		ActionID:    a.actionID,
		DetectionID: a.detectionID,
		ActionType:  "tune_config_high_latency",
		DatabaseID:  a.databaseID,
		Status:      models.StatusCompleted,
		Message:     fmt.Sprintf("Applied %d configuration optimizations. Found %d slow queries requiring code changes.", len(newConfig), len(slowQueries)),
		Changes:     changes,
		CanRollback: true,
	}, nil
}

func (a *TuneConfigAction) Rollback(ctx context.Context) error {
	if a.originalConfig == nil {
		return fmt.Errorf("no original config to rollback to")
	}

	log.Printf("Rolling back config changes for database: %s", a.databaseID)

	err := a.adapter.SetConfig(ctx, a.originalConfig)
	if err != nil {
		return fmt.Errorf("failed to rollback config: %w", err)
	}

	log.Printf("Config rollback complete")
	return nil
}

func (a *TuneConfigAction) Validate(ctx context.Context) error {
	caps := a.adapter.GetCapabilities()

	if !caps.SupportsConfigTuning {
		return fmt.Errorf("database does not support config tuning")
	}

	if !caps.SupportsRuntimeConfigChanges {
		return fmt.Errorf("database does not support runtime config changes (restart required)")
	}

	return nil
}

func (a *TuneConfigAction) GetMetadata() *models.ActionMetadata {
	return &models.ActionMetadata{
		ActionID:     a.actionID,
		ActionType:   "tune_config_high_latency",
		DatabaseID:   a.databaseID,
		DatabaseType: a.databaseType,
	}
}

// Helper: Calculate optimal configuration based on current values
func (a *TuneConfigAction) calculateOptimalConfig(current map[string]string) map[string]string {
	optimal := make(map[string]string)

	// work_mem: Increase from default 4MB to 16MB
	// (helps with sorting and hash operations in queries)
	if current["work_mem"] == "4MB" || current["work_mem"] == "4096kB" {
		optimal["work_mem"] = "16MB"
	}

	// effective_cache_size: Set to 50% of system RAM
	// (helps query planner make better decisions)
	// Default is often too low (4GB)
	if current["effective_cache_size"] != "" {
		optimal["effective_cache_size"] = "8GB"
	}

	// random_page_cost: Assume SSD storage (default 4.0 is for HDD)
	// Lower value makes random reads seem cheaper, improving index usage
	if current["random_page_cost"] == "4" || current["random_page_cost"] == "4.0" {
		optimal["random_page_cost"] = "1.1"
	}

	return optimal
}

// Helper: Get database-specific optimization guide
func (a *TuneConfigAction) getOptimizationGuide() map[string]interface{} {
	guides := map[string]map[string]interface{}{
		"postgres": {
			"title":  "PostgreSQL Query Optimization Guide",
			"url":    "https://www.postgresql.org/docs/current/performance-tips.html",
			"topics": []string{"indexes", "EXPLAIN ANALYZE", "query planning", "configuration tuning"},
			"key_tips": []string{
				"Use EXPLAIN ANALYZE to understand query execution",
				"Add indexes on columns used in WHERE, JOIN, and ORDER BY clauses",
				"Avoid SELECT * - only select columns you need",
				"Consider partial indexes for frequently filtered subsets",
			},
		},
		"mysql": {
			"title":  "MySQL Query Optimization Guide",
			"url":    "https://dev.mysql.com/doc/refman/8.0/en/optimization.html",
			"topics": []string{"indexes", "EXPLAIN", "query optimization"},
		},
		"mongodb": {
			"title":  "MongoDB Query Optimization Guide",
			"url":    "https://docs.mongodb.com/manual/core/query-optimization/",
			"topics": []string{"indexes", "query plans", "aggregation pipeline"},
		},
	}

	guide, exists := guides[a.databaseType]
	if !exists {
		return guides["postgres"]
	}

	return guide
}
