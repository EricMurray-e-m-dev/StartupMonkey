# Decision 016: Table Bloat Detection Strategy

**Date:** 2025-02-05  
**Status:** Accepted

## Context

PostgreSQL uses MVCC (Multi-Version Concurrency Control). When rows are UPDATEd or DELETEd, old versions are marked "dead" but not immediately removed. These accumulate as "table bloat":

- Wastes disk space
- Slows sequential scans (must skip dead tuples)
- Degrades query performance over time

PostgreSQL has autovacuum, but it's often too conservative for high-write workloads.

**Detection Question:**
How do we identify tables that need VACUUM before performance degrades significantly?

## Decision

Implement **ratio-based bloat detection** using `pg_stat_user_tables`:

**Metrics Collected:**
```sql
SELECT relname, n_live_tup, n_dead_tup, last_vacuum, last_autovacuum
FROM pg_stat_user_tables
```

**Bloat Ratio Calculation:**
```go
bloatRatio = deadTuples / liveTuples
```

**Detection Threshold:**
- Default: 10% dead tuples triggers detection
- Configurable via `SetThreshold()`

**Severity Levels:**
| Bloat Ratio | Severity |
|-------------|----------|
| 10-20% | Info |
| 20-30% | Warning |
| 30%+ | Critical |

**Action:**
`vacuum_table` action runs `VACUUM ANALYZE`:
- Reclaims space from dead tuples
- Updates query planner statistics
- Non-blocking, safe for production

## Consequences

**Positive:**
- Proactive bloat management before performance degrades
- Simple, reliable detection using built-in PostgreSQL stats
- VACUUM is safe, non-destructive, no rollback needed
- Automatic re-detection if bloat builds again (correct MAPE-K behaviour)

**Negative:**
- Ratio-based detection may miss tables with low row counts but high absolute bloat
- Doesn't account for table size (10% of 1M rows vs 10% of 1K rows)

**Trade-offs Accepted:**
- Ratio is good enough for MVP; can add absolute thresholds later
- False positives (unnecessary VACUUM) are harmless

## Alternatives Considered

**Absolute Dead Tuple Threshold:**
- Considered: `dead_tuples > 100000`
- Rejected for MVP: Ratio is more universal across table sizes

**Time Since Last Vacuum:**
- Considered: Alert if no vacuum in 7 days
- Rejected: Doesn't account for write activity, table might not need it

**pgstattuple Extension:**
- Considered: More accurate bloat estimation
- Rejected: Another optional extension, adds complexity

**VACUUM FULL:**
- Rejected: Requires exclusive lock, blocks queries, too aggressive for autonomous action