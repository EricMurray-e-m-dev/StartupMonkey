# Decision 017: Execution Mode Strategy

**Date:** 2025-01-15  
**Status:** Accepted

## Context

Autonomous database optimisation is powerful but risky. Different users have different risk tolerances:

- **Startups:** "Just fix it, I don't care how"
- **Enterprises:** "Show me what you'd do, I'll approve it"
- **Cautious users:** "Tell me what's wrong, I'll fix it myself"

A one-size-fits-all approach would either be too aggressive or too passive.

## Decision

Implement **three execution modes** configurable via Dashboard:

**Autonomous Mode:**
- Detections trigger immediate action execution
- No user intervention required
- Best for: Non-critical environments, users who trust the system

**Approval Mode:**
- Detections create actions with `pending_approval` status
- Actions queue in Dashboard for user review
- User can approve, reject, or modify
- Best for: Production environments, cautious users

**Observe Mode:**
- Detections logged and displayed
- No actions created or executed
- Best for: Evaluation period, learning what the system would do

**Implementation:**
```go
switch executionMode {
case models.ModeObserve:
    initialStatus = models.StatusSuggested
case models.ModeApproval:
    initialStatus = models.StatusPendingApproval
default: // autonomous
    initialStatus = models.StatusQueued
    go h.executeAction(action, detection)
}
```

**Storage:**
Execution mode stored in Knowledge service, fetched at detection time.

## Consequences

**Positive:**
- Accommodates different risk tolerances
- Users can start in Observe mode, graduate to Autonomous
- Clear audit trail in all modes
- Same detection logic regardless of mode

**Negative:**
- Approval mode requires user attention (defeats "autonomous" goal)
- Mode is system-wide, not per-action-type

**Future Consideration:**
Per-action-type modes (e.g., "autonomous for VACUUM, approval for index creation") could be added later.

## Alternatives Considered

**Per-Action Risk Levels:**
- Considered: Classify actions as safe/moderate/risky, auto-execute safe ones
- Deferred: Added complexity, can revisit post-MVP

**Dry Run Mode:**
- Considered: Execute but don't commit
- Rejected: Most actions (VACUUM, index creation) don't support dry run

**Scheduled Execution Windows:**
- Considered: Only execute during maintenance windows
- Deferred: Good idea for v2, adds scheduling complexity