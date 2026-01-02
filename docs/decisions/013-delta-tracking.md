# Decision 013: Delta Tracking for Detection Deduplication

**Date:** 2025-11-25  
**Status:** Accepted

## Context

The Analyser runs detection algorithms on every metric snapshot (every 10 seconds). This created a problem:

**Without deduplication:**
1. Metric snapshot arrives with high sequential scans
2. Missing Index Detector fires → Detection published
3. 10 seconds later, next snapshot arrives (issue still exists)
4. Missing Index Detector fires again → Duplicate detection published
5. Executor receives duplicate → Creates index that already exists (or is being created)

This resulted in:
- NATS flooded with duplicate detections
- Executor attempting duplicate actions
- Dashboard showing same issue repeatedly
- No way to know when an issue was actually resolved

## Decision

Implement **Delta Tracking** in the Analyser with Knowledge service integration.

**Detection Lifecycle:**
```
NEW → ACTIVE → RESOLVED
        ↑         │
        └─────────┘ (can recur)
```

**Before Publishing Detection:**
1. Generate deterministic detection ID based on issue type + target (e.g., `missing_index_users_email`)
2. Query Knowledge: "Is there an active detection with this ID?"
3. If YES → Skip publishing (issue already known)
4. If NO → Register detection in Knowledge, then publish

**After Action Completes:**
1. Executor publishes `actions.completed` event
2. Analyser subscribes to `actions.completed`
3. Analyser marks associated detection as RESOLVED in Knowledge
4. Next metric snapshot: if issue recurs, it's treated as new detection

**Knowledge Detection Record:**
```json
{
  "id": "missing_index_users_email",
  "status": "active|resolved",
  "first_detected": "2025-11-25T10:00:00Z",
  "last_seen": "2025-11-25T10:05:00Z",
  "action_id": "action_123"
}
```

## Consequences

**Positive:**
- No duplicate detections published
- Clear detection lifecycle (active → resolved → can recur)
- Feedback loop: Analyser knows when issues are fixed
- Dashboard shows accurate current state
- Executor receives each issue exactly once

**Negative:**
- Additional Knowledge queries on every detection cycle
- Complexity in ID generation (must be deterministic and unique)
- Race condition possible if Analyser restarts mid-cycle

**Trade-offs Accepted:**
- Knowledge query overhead acceptable (Redis is fast, queries are simple)
- Race conditions rare and self-correcting (worst case: one duplicate)

## Alternatives Considered

**In-Memory Deduplication:**
- Rejected: Lost on service restart, no feedback loop

**Time-based Cooldown:**
- Rejected: Arbitrary timing, doesn't account for actual resolution

**Executor-side Deduplication:**
- Rejected: Too late in pipeline, NATS still flooded, other subscribers affected
