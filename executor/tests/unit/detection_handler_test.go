package unit

import (
	"os"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDetectionHandler_HandleDetection(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")

	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID:  "det-123",
		DetectorName: "missing_index_detector",
		Category:     "performance",
		Severity:     "warning",
		DatabaseID:   "test-db",
		Title:        "Missing index on users(id)",
		Description:  "Sequential scans detected on 'users' table due to missing index.",
		ActionType:   "create_index",
		ActionMetaData: map[string]interface{}{
			"table_name":  "users",
			"column_name": "id",
		},
		Timestamp: time.Now().Unix(),
	}

	result, err := h.HandleDetection(detection)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ActionID)
	assert.Equal(t, "det-123", result.DetectionID)
	assert.Equal(t, "create_index", result.ActionType)
	assert.Equal(t, "test-db", result.DatabaseID)
	assert.Equal(t, models.StatusQueued, result.Status)
	assert.NotZero(t, result.CreatedAt)
	assert.Nil(t, result.Completed)
}

func TestDetectionHandler_GetActionStatus_Success(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID: "det123",
		ActionType:  "create_index",
		ActionMetaData: map[string]interface{}{
			"table_name":  "users",
			"column_name": "id",
		},
		DatabaseID: "test-db",
		Timestamp:  time.Now().Unix(),
	}

	result, err := h.HandleDetection(detection)
	assert.NoError(t, err)

	retrieved, err := h.GetActionStatus(result.ActionID)
	assert.NoError(t, err)

	assert.NotNil(t, retrieved)
	assert.Equal(t, result.ActionID, retrieved.ActionID)
	assert.Equal(t, "det123", retrieved.DetectionID)
	assert.Equal(t, models.StatusQueued, retrieved.Status)
}

func TestDetectionHandler_GetActionStatus_NotFound(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	retrieved, err := h.GetActionStatus("fake-action-id")
	assert.Error(t, err)

	assert.Nil(t, retrieved)
	assert.Contains(t, err.Error(), "action not found")
}

func TestDetectionHandler_ListPendingActions_NoFilter(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	detection1 := &models.Detection{
		DetectionID: "det-1",
		ActionType:  "create_index",
		ActionMetaData: map[string]interface{}{
			"table_name":  "users",
			"column_name": "id",
		},
		Timestamp: time.Now().Unix(),
	}

	detection2 := &models.Detection{
		DetectionID: "det-2",
		ActionType:  "deploy_pgbouncer",
		Timestamp:   time.Now().Unix(),
	}

	_, err := h.HandleDetection(detection1)
	assert.NoError(t, err)

	_, err = h.HandleDetection(detection2)
	assert.NoError(t, err)

	actions, err := h.ListPendingActions("")
	assert.NoError(t, err)

	assert.Len(t, actions, 2)
}

func TestDetectionHandler_ListPendingActions_WithFilter(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID: "det-1",
		ActionType:  "create_index",
		ActionMetaData: map[string]interface{}{
			"table_name":  "users",
			"column_name": "id",
		},
		Timestamp: time.Now().Unix(),
	}

	result, err := h.HandleDetection(detection)
	assert.NoError(t, err)

	actions, err := h.ListPendingActions(models.StatusQueued)
	assert.NoError(t, err)

	assert.Len(t, actions, 1)
	assert.Equal(t, result.ActionID, actions[0].ActionID)
}

func TestDetectionHandler_ListPendingActions_EmptyList(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	actions, err := h.ListPendingActions("")
	assert.NoError(t, err)

	assert.Empty(t, actions)
}

func TestDetectionHandler_ConcurrentHandling(t *testing.T) {
	os.Setenv("DB_CONNECTION_STRING", "postgresql://ericmurray@localhost:5432/bad_performance_test")
	h := handler.NewDetectionHandler()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(index int) {
			detection := &models.Detection{
				DetectionID: "det-concurrent",
				ActionType:  "create_index",
				ActionMetaData: map[string]interface{}{
					"table_name":  "users",
					"column_name": "id",
				},
				Timestamp: time.Now().Unix(),
			}

			_, err := h.HandleDetection(detection)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	actions, err := h.ListPendingActions("")
	assert.NoError(t, err)
	assert.Len(t, actions, 10)
}
