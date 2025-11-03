package unit

import (
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDetectionHandler_HandleDetection(t *testing.T) {
	t.Skip("Skipping tests that require db connection for now")
	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID:  "det-123",
		DetectorName: "connection_pool_exhaustion",
		Category:     "connection",
		Severity:     "critical",
		DatabaseID:   "test-db",
		Title:        "Connection pool at 95%",
		Description:  "Pool exhausted",
		ActionType:   "deploy_connection_pooler",
		Timestamp:    time.Now().Unix(),
	}

	result, err := h.HandleDetection(detection)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ActionID)
	assert.Equal(t, "det-123", result.DetectionID)
	assert.Equal(t, "deploy_connection_pooler", result.ActionType)
	assert.Equal(t, "test-db", result.DatabaseID)
	assert.Equal(t, models.StatusQueued, result.Status)
	assert.NotZero(t, result.CreatedAt)
	assert.Nil(t, result.Completed)
}

func TestDetectionHandler_GetActionStatus_Success(t *testing.T) {
	t.Skip("Skipping tests that require db connection for now")
	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID: "det123",
		ActionType:  "create_index",
		DatabaseID:  "test-db",
		Timestamp:   time.Now().Unix(),
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
	h := handler.NewDetectionHandler()

	retrieved, err := h.GetActionStatus("fake-action-id")
	assert.Error(t, err)

	assert.Nil(t, retrieved)
	assert.Contains(t, err.Error(), "action not found")
}

func TestDetectionHandler_ListPendingActions_NoFilter(t *testing.T) {
	t.Skip("Skipping tests that require db connection for now")
	h := handler.NewDetectionHandler()

	detection1 := &models.Detection{
		DetectionID: "det-1",
		ActionType:  "create_index",
		Timestamp:   time.Now().Unix(),
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
	t.Skip("Skipping tests that require db connection for now")
	h := handler.NewDetectionHandler()

	detection := &models.Detection{
		DetectionID: "det-1",
		ActionType:  "create_index",
		Timestamp:   time.Now().Unix(),
	}

	result, err := h.HandleDetection(detection)
	assert.NoError(t, err)

	actions, err := h.ListPendingActions(models.StatusQueued)
	assert.NoError(t, err)

	assert.Len(t, actions, 1)
	assert.Equal(t, result.ActionID, actions[0].ActionID)
}

func TestDetectionHandler_ListPendingActions_EmptyList(t *testing.T) {
	h := handler.NewDetectionHandler()

	actions, err := h.ListPendingActions("")
	assert.NoError(t, err)

	assert.Empty(t, actions)
}

func TestDetectionHandler_ConcurrentHandling(t *testing.T) {
	t.Skip("Skipping tests that require db connection for now")
	h := handler.NewDetectionHandler()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(index int) {
			detection := &models.Detection{
				DetectionID: "det-concurrent",
				ActionType:  "create_index",
				Timestamp:   time.Now().Unix(),
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
