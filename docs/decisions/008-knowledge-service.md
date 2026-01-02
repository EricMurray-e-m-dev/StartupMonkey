# Decision 008: Knowledge Service Architecture

**Date:** 2025-11-18  
**Status:** Accepted

## Context

As StartupMonkey grew, several problems emerged with distributed state management:

1. **Detection Duplication:** Analyser was publishing the same detection repeatedly because it had no memory of previous detections
2. **Authentication Chaos:** Executor needed database credentials to apply actions, but credentials were hardcoded per service
3. **No Feedback Loop:** When Executor completed an action, Analyser had no way to know the issue was resolved
4. **Action Tracking:** No central record of what actions had been applied, making rollback coordination difficult

Each service was managing its own state, leading to inconsistency and tight coupling through shared environment variables.

## Decision

Introduce a dedicated **Knowledge Service** as the centralised state repository for the entire system.

**Architecture:**
```
Collector  ──┐
Analyser   ──┼── gRPC ──► Knowledge Service ──► Redis
Executor   ──┤
Dashboard  ──┘
```

**Responsibilities:**
- **Database Registry:** Store connection info for monitored databases (registered by Collector on startup)
- **Detection Registry:** Track active/resolved detections (prevents duplication)
- **Action Registry:** Record all actions with status (pending/completed/failed/rolled_back)
- **System Stats:** Aggregated metrics for Dashboard

**Technology Choice: Redis**
- Key-value model fits our access patterns (lookup by ID)
- Fast reads/writes for real-time operation
- Simple deployment (single container)
- TTL support for automatic cleanup of old records

## Consequences

**Positive:**
- Single source of truth for system state
- Collector registers DB credentials once, Executor retrieves them (no hardcoding)
- Detection deduplication solved (check Knowledge before publishing)
- Feedback loop enabled (Executor updates Knowledge, Analyser subscribes to changes)
- Dashboard can query Knowledge for current state
- Clean separation of concerns

**Negative:**
- Additional service to deploy and monitor
- All services now depend on Knowledge (single point of failure)
- Network latency for state operations (mitigated by Redis speed)
- Added complexity in service startup order (Knowledge must be up first)

**Trade-offs Accepted:**
- Dependency on Knowledge acceptable for the benefits gained
- Redis is simple enough that operational overhead is minimal

## Alternatives Considered

**Shared PostgreSQL Database:**
- Rejected: Heavier than needed, SQL overhead for simple key-value operations

**Each Service Manages Own State:**
- Rejected: Already tried this, led to duplication and coordination problems

**Distributed State (etcd/Consul):**
- Rejected: Overkill for our scale, adds operational complexity

**In-Memory State with Event Sourcing:**
- Rejected: Loses state on restart, complex to implement correctly
