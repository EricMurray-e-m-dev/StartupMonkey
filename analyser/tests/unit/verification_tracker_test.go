package unit

import (
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/verification"
	"github.com/stretchr/testify/assert"
)

func TestNewTracker(t *testing.T) {
	tracker := verification.NewTracker(3, nil, nil)

	assert.NotNil(t, tracker)
	assert.Equal(t, 0, tracker.GetPendingCount())
}

func TestNewTracker_DefaultCycles(t *testing.T) {
	// When requiredCycles is 0 or negative, should use default (3)
	tracker := verification.NewTracker(0, nil, nil)

	assert.NotNil(t, tracker)
}

func TestAddPendingVerification(t *testing.T) {
	tracker := verification.NewTracker(3, nil, nil)

	tracker.AddPendingVerification(
		"testdb:missing_index:posts.user_id",
		"detection-123",
		"action-456",
		"create_index",
		"testdb",
	)

	assert.Equal(t, 1, tracker.GetPendingCount())
	assert.True(t, tracker.IsPendingVerification("testdb:missing_index:posts.user_id"))
	assert.False(t, tracker.IsPendingVerification("nonexistent-key"))
}

func TestOnDetectionFired_NoExistingVerification(t *testing.T) {
	tracker := verification.NewTracker(3, nil, nil)

	// Should return false when no pending verification exists
	result := tracker.OnDetectionFired("nonexistent-key")

	assert.False(t, result)
}

func TestOnDetectionFired_GracePeriod(t *testing.T) {
	rollbackCalled := false
	tracker := verification.NewTracker(3, func(req *verification.RollbackRequest) {
		rollbackCalled = true
	}, nil)

	tracker.AddPendingVerification(
		"testdb:missing_index:posts.user_id",
		"detection-123",
		"action-456",
		"create_index",
		"testdb",
	)

	// Fire detection immediately (0 cycles elapsed)
	// Should suppress detection but NOT trigger rollback
	result := tracker.OnDetectionFired("testdb:missing_index:posts.user_id")

	assert.True(t, result, "Should return true to suppress detection")
	assert.False(t, rollbackCalled, "Should NOT trigger rollback during grace period")
	assert.True(t, tracker.IsPendingVerification("testdb:missing_index:posts.user_id"), "Should still be pending")
}

func TestOnDetectionFired_AfterGracePeriod(t *testing.T) {
	var receivedRequest *verification.RollbackRequest
	tracker := verification.NewTracker(3, func(req *verification.RollbackRequest) {
		receivedRequest = req
	}, nil)

	tracker.AddPendingVerification(
		"testdb:missing_index:posts.user_id",
		"detection-123",
		"action-456",
		"create_index",
		"testdb",
	)

	// Simulate one collection cycle passing
	tracker.OnCollectionCycle()

	// Now fire detection (1 cycle elapsed, past grace period)
	result := tracker.OnDetectionFired("testdb:missing_index:posts.user_id")

	assert.True(t, result, "Should return true")
	assert.NotNil(t, receivedRequest, "Should trigger rollback callback")
	assert.Equal(t, "action-456", receivedRequest.ActionID)
	assert.Equal(t, "detection-123", receivedRequest.DetectionID)
	assert.Equal(t, "create_index", receivedRequest.ActionType)
	assert.Equal(t, "testdb", receivedRequest.DatabaseID)
	assert.Equal(t, "Issue re-detected after action completion", receivedRequest.Reason)
	assert.False(t, tracker.IsPendingVerification("testdb:missing_index:posts.user_id"), "Should be removed")
}

func TestOnCollectionCycle_IncrementsCycles(t *testing.T) {
	tracker := verification.NewTracker(5, nil, nil)

	tracker.AddPendingVerification(
		"testdb:missing_index:posts.user_id",
		"detection-123",
		"action-456",
		"create_index",
		"testdb",
	)

	// Run 2 cycles
	tracker.OnCollectionCycle()
	tracker.OnCollectionCycle()

	// Should still be pending (need 5 cycles)
	assert.True(t, tracker.IsPendingVerification("testdb:missing_index:posts.user_id"))
	assert.Equal(t, 1, tracker.GetPendingCount())
}

func TestOnCollectionCycle_VerifiedAfterRequiredCycles(t *testing.T) {
	var verifiedDetectionID string
	tracker := verification.NewTracker(3, nil, func(detectionID string) {
		verifiedDetectionID = detectionID
	})

	tracker.AddPendingVerification(
		"testdb:missing_index:posts.user_id",
		"detection-123",
		"action-456",
		"create_index",
		"testdb",
	)

	// Run 3 cycles
	tracker.OnCollectionCycle()
	tracker.OnCollectionCycle()
	tracker.OnCollectionCycle()

	assert.Equal(t, "detection-123", verifiedDetectionID, "Should call verified callback")
	assert.False(t, tracker.IsPendingVerification("testdb:missing_index:posts.user_id"), "Should be removed")
	assert.Equal(t, 0, tracker.GetPendingCount())
}

func TestOnCollectionCycle_MultipleVerifications(t *testing.T) {
	verifiedCount := 0
	tracker := verification.NewTracker(2, nil, func(detectionID string) {
		verifiedCount++
	})

	tracker.AddPendingVerification("key1", "det-1", "action-1", "create_index", "db1")
	tracker.AddPendingVerification("key2", "det-2", "action-2", "create_index", "db2")
	tracker.AddPendingVerification("key3", "det-3", "action-3", "create_index", "db3")

	assert.Equal(t, 3, tracker.GetPendingCount())

	// Run 2 cycles - all should verify
	tracker.OnCollectionCycle()
	tracker.OnCollectionCycle()

	assert.Equal(t, 3, verifiedCount, "All 3 should be verified")
	assert.Equal(t, 0, tracker.GetPendingCount())
}

func TestGetPendingVerifications(t *testing.T) {
	tracker := verification.NewTracker(3, nil, nil)

	tracker.AddPendingVerification("key1", "det-1", "action-1", "create_index", "db1")
	tracker.AddPendingVerification("key2", "det-2", "action-2", "deploy_pgbouncer", "db2")

	pending := tracker.GetPendingVerifications()

	assert.Len(t, pending, 2)

	// Verify it returns copies (modifying shouldn't affect tracker)
	pending[0].ActionID = "modified"
	assert.True(t, tracker.IsPendingVerification("key1"), "Original should be unaffected")
}

func TestRollbackNotTriggeredOnVerification(t *testing.T) {
	rollbackCalled := false
	verifiedCalled := false

	tracker := verification.NewTracker(2,
		func(req *verification.RollbackRequest) {
			rollbackCalled = true
		},
		func(detectionID string) {
			verifiedCalled = true
		},
	)

	tracker.AddPendingVerification("key1", "det-1", "action-1", "create_index", "db1")

	// Run cycles to trigger verification
	tracker.OnCollectionCycle()
	tracker.OnCollectionCycle()

	assert.True(t, verifiedCalled, "Verified callback should be called")
	assert.False(t, rollbackCalled, "Rollback should NOT be called on successful verification")
}

func TestOverwriteExistingVerification(t *testing.T) {
	tracker := verification.NewTracker(3, nil, nil)

	tracker.AddPendingVerification("same-key", "det-1", "action-1", "create_index", "db1")
	tracker.OnCollectionCycle() // 1 cycle

	// Overwrite with new verification (same key)
	tracker.AddPendingVerification("same-key", "det-2", "action-2", "create_index", "db1")

	// Should still be 1 pending (overwritten, not added)
	assert.Equal(t, 1, tracker.GetPendingCount())

	pending := tracker.GetPendingVerifications()
	assert.Equal(t, "action-2", pending[0].ActionID, "Should have new action ID")
	assert.Equal(t, 0, pending[0].CyclesElapsed, "Cycles should reset")
}
