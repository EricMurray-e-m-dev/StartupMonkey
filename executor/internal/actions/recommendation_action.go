package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

// RecommendationAction generates optimization recommendations without executing changes.
// This action type is used for risky optimizations that require user approval or code changes.
type RecommendationAction struct {
	actionID     string
	detectionID  string
	databaseID   string
	databaseType string

	// Recommendations with risk levels
	recommendations []Recommendation
}

// Recommendation represents a single optimization suggestion with risk assessment.
type Recommendation struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	RiskLevel          string   `json:"risk_level"` // "safe", "medium", "advanced"
	Steps              []string `json:"steps"`
	RequiresRestart    bool     `json:"requires_restart"`
	RequiresCodeChange bool     `json:"requires_code_change"`

	// Optional: Link to deployable action (e.g., "deploy_redis")
	DeployableActionType string `json:"deployable_action_type,omitempty"`
}

// NewRecommendationAction creates a new recommendation action based on detection metadata.
func NewRecommendationAction(
	actionID string,
	detectionID string,
	databaseID string,
	databaseType string,
	detectionMetadata map[string]interface{},
) *RecommendationAction {
	recommendations := buildRecommendations(databaseType, detectionMetadata)

	return &RecommendationAction{
		actionID:        actionID,
		detectionID:     detectionID,
		databaseID:      databaseID,
		databaseType:    databaseType,
		recommendations: recommendations,
	}
}

// Execute generates and stores recommendations without making any changes.
func (a *RecommendationAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	startTime := time.Now()

	// Generate recommendations based on database type and detection
	result := &models.ActionResult{
		ActionID:        a.actionID,
		DetectionID:     a.detectionID,
		ActionType:      "recommendation",
		DatabaseID:      a.databaseID,
		Status:          models.StatusCompleted,
		Message:         fmt.Sprintf("Generated %d recommendations for %s", len(a.recommendations), a.databaseType),
		CreatedAt:       startTime,
		Started:         &startTime,
		Completed:       &startTime,
		ExecutionTimeMs: 0, // Instant
		Changes: map[string]interface{}{
			"recommendations": a.recommendations,
			"database_type":   a.databaseType,
		},
		CanRollback: false, // Nothing to rollback - no changes made
	}

	return result, nil
}

// Rollback is not applicable for recommendation actions.
func (a *RecommendationAction) Rollback(ctx context.Context) error {
	return fmt.Errorf("recommendation actions cannot be rolled back (no changes were made)")
}

// Validate checks that recommendations are properly structured.
func (a *RecommendationAction) Validate(ctx context.Context) error {
	if len(a.recommendations) == 0 {
		return fmt.Errorf("no recommendations generated")
	}

	for i, rec := range a.recommendations {
		if rec.Title == "" {
			return fmt.Errorf("recommendation %d missing title", i)
		}
		if rec.RiskLevel == "" {
			return fmt.Errorf("recommendation %d missing risk level", i)
		}
	}

	return nil
}

// GetMetadata returns action metadata.
func (a *RecommendationAction) GetMetadata() *models.ActionMetadata {
	return &models.ActionMetadata{
		ActionID:     a.actionID,
		ActionType:   "recommendation",
		DatabaseID:   a.databaseID,
		DatabaseType: a.databaseType,
		CreatedAt:    time.Now(),
	}
}

// buildRecommendations generates database-specific recommendations from detection metadata.
func buildRecommendations(databaseType string, metadata map[string]interface{}) []Recommendation {
	var recommendations []Recommendation

	// Extract safe and advanced options from detection metadata
	if safeOption, ok := metadata["safe_option"].(map[string]interface{}); ok {
		rec := Recommendation{
			Title:           getStringFromMap(safeOption, "title", "Increase Database Cache"),
			Description:     getStringFromMap(safeOption, "description", "Increase database cache size to improve performance"),
			RiskLevel:       "safe",
			Steps:           getStepsFromMap(safeOption),
			RequiresRestart: getBoolFromMap(safeOption, "requires_restart", false),
		}
		recommendations = append(recommendations, rec)
	}

	if advancedOption, ok := metadata["advanced_option"].(map[string]interface{}); ok {
		rec := Recommendation{
			Title:                getStringFromMap(advancedOption, "title", "Deploy Cache Layer"),
			Description:          getStringFromMap(advancedOption, "description", "Deploy Redis for application-level caching"),
			RiskLevel:            "advanced",
			Steps:                getStepsFromMap(advancedOption),
			RequiresCodeChange:   true,
			DeployableActionType: getStringFromMap(advancedOption, "deployable_action", ""),
		}
		recommendations = append(recommendations, rec)
	}

	// If no recommendations in metadata, generate default based on database type
	if len(recommendations) == 0 {
		recommendations = append(recommendations, getDefaultRecommendation(databaseType))
	}

	return recommendations
}

// Helper functions for extracting data from metadata maps
func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func getBoolFromMap(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getStepsFromMap(m map[string]interface{}) []string {
	if stepsInterface, ok := m["steps"].([]interface{}); ok {
		steps := make([]string, 0, len(stepsInterface))
		for _, s := range stepsInterface {
			if str, ok := s.(string); ok {
				steps = append(steps, str)
			}
		}
		return steps
	}
	return []string{}
}

// getDefaultRecommendation provides fallback recommendations if metadata is missing.
func getDefaultRecommendation(databaseType string) Recommendation {
	switch databaseType {
	case "postgres", "postgresql":
		return Recommendation{
			Title:           "Increase PostgreSQL Cache",
			Description:     "Increase shared_buffers to improve query performance",
			RiskLevel:       "safe",
			RequiresRestart: true,
			Steps: []string{
				"Edit postgresql.conf",
				"Set shared_buffers = 256MB",
				"Restart PostgreSQL service",
				"Monitor cache hit rate in Dashboard",
			},
		}
	case "mysql":
		return Recommendation{
			Title:           "Increase MySQL Buffer Pool",
			Description:     "Increase innodb_buffer_pool_size to improve performance",
			RiskLevel:       "safe",
			RequiresRestart: true,
			Steps: []string{
				"Edit my.cnf",
				"Set innodb_buffer_pool_size = 512MB",
				"Restart MySQL service",
				"Monitor cache hit rate",
			},
		}
	default:
		return Recommendation{
			Title:       "Review Cache Configuration",
			Description: "Manually review and optimize cache settings for your database",
			RiskLevel:   "medium",
			Steps: []string{
				"Review database documentation",
				"Identify cache-related configuration",
				"Increase cache size appropriately",
			},
		}
	}
}
