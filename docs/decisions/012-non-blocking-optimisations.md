# Decision 012: Non-blocking Optimisation Strategy

**Date:** 2025-11-07  
**Status:** Accepted

## Context

StartupMonkey applies optimisations to production databases. A key principle is **"do no harm"** - optimisations must not cause downtime or degrade performance while being applied.

**Risky operations to avoid:**
- `CREATE INDEX` - Locks table for writes until complete (can take minutes on large tables)
- Configuration changes requiring restart
- Any operation that blocks application queries

**Target users (solo devs, startups) likely:**
- Don't have maintenance windows
- Can't afford any downtime
- May not understand the risk of certain operations

## Decision

All optimisation actions must be **non-blocking** by design.

**Index Creation:**
```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_name ON table (column);
```
- PostgreSQL builds index in background
- Table remains fully writable during creation
- Takes longer but zero downtime
- Requires extra disk space temporarily

**Connection Pooling (PgBouncer):**
- Deploy as separate container
- User switches connection string (port 5432 → 6432)
- No changes to existing database
- Instant rollback: just switch back to direct connection

**Caching (Redis):**
- Deploy as separate container
- Application integration required (not automatic)
- No impact on existing database
- Instant rollback: remove container

**Configuration Tuning:**
- Only parameters that don't require restart
- `effective_cache_size` - Runtime changeable
- Store original value for rollback
- Apply with `ALTER SYSTEM` + `pg_reload_conf()`

## Consequences

**Positive:**
- Zero downtime during optimisations
- Safe for production use
- Builds trust with users (system won't break their app)
- Aligns with autonomous operation principle

**Negative:**
- `CREATE INDEX CONCURRENTLY` takes 2-3x longer than regular
- `CREATE INDEX CONCURRENTLY` can fail and leave invalid index (needs cleanup)
- Some optimisations not possible (e.g., changing `shared_buffers` requires restart)
- Container deployments require Docker socket access

**Trade-offs Accepted:**
- Slower index creation is acceptable for safety
- Excluding restart-required config changes limits optimisation scope but maintains safety

## Alternatives Considered

**Regular CREATE INDEX with Warning:**
- Rejected: Users might not understand the warning, autonomous system shouldn't require user judgement

**Scheduled Maintenance Windows:**
- Rejected: Target users don't have maintenance windows, adds complexity

**Read Replica Optimisation:**
- Rejected: Assumes infrastructure our target users don't have
