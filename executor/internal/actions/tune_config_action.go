package actions

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	// 3. Apply configuration changes (only if there are changes)
	changesMade := len(newConfig) > 0
	if changesMade {
		err = a.adapter.SetConfig(ctx, newConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to apply config changes: %w", err)
		}

		log.Printf("Applied config changes: %+v", newConfig)
		a.appliedChanges = newConfig
	} else {
		log.Printf("No configuration changes needed - all parameters already optimal")
	}

	// 4. Get slow queries for educational component
	slowQueries, err := a.adapter.GetSlowQueries(ctx, 500.0, 5)
	if err != nil {
		log.Printf("Warning: failed to retrieve slow queries: %v", err)
		slowQueries = []database.SlowQuery{}
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

	// Build message based on whether changes were made
	var message string
	if changesMade {
		message = fmt.Sprintf("Applied %d configuration optimizations. Found %d slow queries requiring code changes.", len(newConfig), len(slowQueries))
	} else {
		message = fmt.Sprintf("Configuration already optimal. Found %d slow queries requiring code changes.", len(slowQueries))
	}

	return &models.ActionResult{
		ActionID:    a.actionID,
		DetectionID: a.detectionID,
		ActionType:  "tune_config_high_latency",
		DatabaseID:  a.databaseID,
		Status:      models.StatusCompleted,
		Message:     message,
		Changes:     changes,
		CanRollback: changesMade, // Only allow rollback if we actually made changes
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
	if currentVal := current["work_mem"]; currentVal == "4MB" || currentVal == "4096kB" {
		optimal["work_mem"] = "16MB"
	}

	// effective_cache_size: Increase if currently at default (4GB or less)
	if currentVal := current["effective_cache_size"]; currentVal != "" {
		// Parse current value (handle GB/MB)
		if currentVal == "4GB" || currentVal == "4096MB" || strings.HasPrefix(currentVal, "1GB") || strings.HasPrefix(currentVal, "2GB") || strings.HasPrefix(currentVal, "512MB") {
			optimal["effective_cache_size"] = "8GB"
		}
		// If it's already 8GB or higher, don't change it
	}

	// random_page_cost: Lower from HDD default (4.0) to SSD (1.1)
	if currentVal := current["random_page_cost"]; currentVal == "4" || currentVal == "4.0" {
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
