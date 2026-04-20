package unit

import (
	"context"
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRecommendationAction_ExecuteWithMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"safe_option": map[string]interface{}{
			"title":            "Increase Cache Size",
			"description":      "Boost cache to 512MB",
			"requires_restart": true,
			"steps":            []interface{}{"Edit config", "Restart service"},
		},
		"advanced_option": map[string]interface{}{
			"title":             "Deploy Redis",
			"description":       "Add Redis caching layer",
			"deployable_action": "deploy_redis",
			"steps":             []interface{}{"Install Redis", "Configure app"},
		},
	}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.False(t, result.CanRollback)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.Len(t, recommendations, 2)
	assert.Equal(t, "Increase Cache Size", recommendations[0].Title)
	assert.Equal(t, "safe", recommendations[0].RiskLevel)
	assert.Equal(t, "Deploy Redis", recommendations[1].Title)
	assert.Equal(t, "advanced", recommendations[1].RiskLevel)
}

func TestRecommendationAction_ExecuteWithDefaultPostgres(t *testing.T) {
	// Empty metadata triggers default recommendation
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCompleted, result.Status)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.Len(t, recommendations, 1)
	assert.Equal(t, "Increase PostgreSQL Cache", recommendations[0].Title)
	assert.Equal(t, "safe", recommendations[0].RiskLevel)
	assert.True(t, recommendations[0].RequiresRestart)
}

func TestRecommendationAction_ExecuteWithDefaultMySQL(t *testing.T) {
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "mysql", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.Len(t, recommendations, 1)
	assert.Equal(t, "Increase MySQL Buffer Pool", recommendations[0].Title)
}

func TestRecommendationAction_ExecuteWithDefaultUnknownDB(t *testing.T) {
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "unknown", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.Len(t, recommendations, 1)
	assert.Equal(t, "Review Cache Configuration", recommendations[0].Title)
	assert.Equal(t, "medium", recommendations[0].RiskLevel)
}

func TestRecommendationAction_ValidateSuccess(t *testing.T) {
	metadata := map[string]interface{}{
		"safe_option": map[string]interface{}{
			"title":       "Test Recommendation",
			"description": "Test description",
		},
	}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	err := action.Validate(context.Background())

	assert.NoError(t, err)
}

func TestRecommendationAction_ValidateWithDefaults(t *testing.T) {
	// Empty metadata gets default recommendation which has title and risk_level
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	err := action.Validate(context.Background())

	assert.NoError(t, err)
}

func TestRecommendationAction_RollbackReturnsError(t *testing.T) {
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	err := action.Rollback(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be rolled back")
}

func TestRecommendationAction_GetMetadata(t *testing.T) {
	metadata := map[string]interface{}{}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result := action.GetMetadata()

	assert.Equal(t, "action-1", result.ActionID)
	assert.Equal(t, "recommendation", result.ActionType)
	assert.Equal(t, "test-db", result.DatabaseID)
	assert.Equal(t, "postgres", result.DatabaseType)
}

func TestRecommendationAction_SafeOptionExtractsSteps(t *testing.T) {
	metadata := map[string]interface{}{
		"safe_option": map[string]interface{}{
			"title": "Test",
			"steps": []interface{}{"Step 1", "Step 2", "Step 3"},
		},
	}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.Len(t, recommendations[0].Steps, 3)
	assert.Equal(t, "Step 1", recommendations[0].Steps[0])
}

func TestRecommendationAction_AdvancedOptionSetsCodeChange(t *testing.T) {
	metadata := map[string]interface{}{
		"advanced_option": map[string]interface{}{
			"title":             "Deploy Cache",
			"deployable_action": "deploy_redis",
		},
	}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)

	recommendations := result.Changes["recommendations"].([]actions.Recommendation)
	assert.True(t, recommendations[0].RequiresCodeChange)
	assert.Equal(t, "deploy_redis", recommendations[0].DeployableActionType)
}

func TestRecommendationAction_MessageIncludesCount(t *testing.T) {
	metadata := map[string]interface{}{
		"safe_option": map[string]interface{}{
			"title": "Rec 1",
		},
		"advanced_option": map[string]interface{}{
			"title": "Rec 2",
		},
	}

	action := actions.NewRecommendationAction("action-1", "detection-1", "test-db", "postgres", metadata)

	result, err := action.Execute(context.Background())

	assert.NoError(t, err)
	assert.Contains(t, result.Message, "2 recommendations")
}
