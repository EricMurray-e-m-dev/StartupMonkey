package verification

import (
	"log"
	"sync"
	"time"
)

const (
	// Number of collection cycles to wait before confirming action worked
	DefaultVerificationCycles = 3

	// Minimum cycles before rollback can trigger (grace period)
	MinCyclesBeforeRollback = 1

	// Max time to wait for verification before giving up (not rolling back, just abandoning check)
	MaxVerificationTime = 10 * time.Minute
)

// PendingVerification tracks an action awaiting verification
type PendingVerification struct {
	DetectionKey  string
	DetectionID   string
	ActionID      string
	ActionType    string
	DatabaseID    string
	CompletedAt   time.Time
	CyclesElapsed int
}

// RollbackRequest is published when verification fails
type RollbackRequest struct {
	ActionID    string `json:"action_id"`
	DetectionID string `json:"detection_id"`
	ActionType  string `json:"action_type"`
	DatabaseID  string `json:"database_id"`
	Reason      string `json:"reason"`
	Timestamp   int64  `json:"timestamp"`
}

// Tracker manages pending action verifications
type Tracker struct {
	pending          map[string]*PendingVerification // keyed by DetectionKey
	mu               sync.RWMutex
	requiredCycles   int
	onRollbackNeeded func(request *RollbackRequest)
	onVerified       func(detectionID string)
}

// NewTracker creates a new verification tracker
func NewTracker(requiredCycles int, onRollbackNeeded func(*RollbackRequest), onVerified func(string)) *Tracker {
	if requiredCycles <= 0 {
		requiredCycles = DefaultVerificationCycles
	}

	return &Tracker{
		pending:          make(map[string]*PendingVerification),
		requiredCycles:   requiredCycles,
		onRollbackNeeded: onRollbackNeeded,
		onVerified:       onVerified,
	}
}

// AddPendingVerification adds an action to be verified
func (t *Tracker) AddPendingVerification(detectionKey, detectionID, actionID, actionType, databaseID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pending[detectionKey] = &PendingVerification{
		DetectionKey:  detectionKey,
		DetectionID:   detectionID,
		ActionID:      actionID,
		ActionType:    actionType,
		DatabaseID:    databaseID,
		CompletedAt:   time.Now(),
		CyclesElapsed: 0,
	}

	log.Printf("[Verification] Added pending verification for action %s (detection: %s, key: %s)",
		actionID, detectionID, detectionKey)
}

// OnDetectionFired is called when a detection would fire
// Returns true if this detection has a pending verification (suppresses the detection)
func (t *Tracker) OnDetectionFired(detectionKey string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	pv, exists := t.pending[detectionKey]
	if !exists {
		return false
	}

	// Grace period - don't rollback on first cycle, metrics need time to reflect the fix
	if pv.CyclesElapsed < MinCyclesBeforeRollback {
		log.Printf("[Verification] Detection %s re-fired but in grace period (cycle %d < %d), suppressing",
			detectionKey, pv.CyclesElapsed, MinCyclesBeforeRollback)
		return true // Suppress detection but don't rollback yet
	}

	// Same issue detected again after grace period - action didn't help
	log.Printf("[Verification] Action %s did not resolve issue (detection key: %s fired again after %d cycles)",
		pv.ActionID, detectionKey, pv.CyclesElapsed)

	// Trigger rollback
	if t.onRollbackNeeded != nil {
		t.onRollbackNeeded(&RollbackRequest{
			ActionID:    pv.ActionID,
			DetectionID: pv.DetectionID,
			ActionType:  pv.ActionType,
			DatabaseID:  pv.DatabaseID,
			Reason:      "Issue re-detected after action completion",
			Timestamp:   time.Now().Unix(),
		})
	}

	// Remove from pending
	delete(t.pending, detectionKey)

	return true
}

// OnCollectionCycle is called after each metrics collection
// It increments cycle counts and marks verified actions as resolved
func (t *Tracker) OnCollectionCycle() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for key, pv := range t.pending {
		// Check for timeout
		if now.Sub(pv.CompletedAt) > MaxVerificationTime {
			log.Printf("[Verification] Verification timeout for action %s, abandoning check", pv.ActionID)
			toRemove = append(toRemove, key)
			continue
		}

		pv.CyclesElapsed++

		// Check if enough cycles passed without re-detection
		if pv.CyclesElapsed >= t.requiredCycles {
			log.Printf("[Verification] Action %s verified after %d cycles - marking resolved",
				pv.ActionID, pv.CyclesElapsed)

			// Trigger resolved callback
			if t.onVerified != nil {
				t.onVerified(pv.DetectionID)
			}

			toRemove = append(toRemove, key)
		}
	}

	// Clean up
	for _, key := range toRemove {
		delete(t.pending, key)
	}
}

// IsPendingVerification checks if a detection key has a pending verification
func (t *Tracker) IsPendingVerification(detectionKey string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, exists := t.pending[detectionKey]
	return exists
}

// GetPendingCount returns the number of pending verifications
func (t *Tracker) GetPendingCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.pending)
}

// GetPendingVerifications returns a copy of all pending verifications
func (t *Tracker) GetPendingVerifications() []*PendingVerification {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*PendingVerification, 0, len(t.pending))
	for _, pv := range t.pending {
		// Return a copy
		copy := *pv
		result = append(result, &copy)
	}
	return result
}
