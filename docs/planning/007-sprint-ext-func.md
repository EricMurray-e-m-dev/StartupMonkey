# Sprint 007: PostgreSQL Detector Depth

**Sprint:** 007
**Dates:** Jan 26 - Feb 6 (2 weeks)
**Theme:** Expand PostgreSQL-specific detectors and evaluation infrastructure

## Primary Objective

Add **PostgreSQL-specific detectors** that demonstrate deep database expertise for dissertation evaluation. Focus on detectors with clear before/after metrics.

## Objectives

1. **Graceful degradation for extensions** - pg_stat_statements auto-enable with fallback
2. **Table bloat detection + VACUUM action** - Dead tuple monitoring
3. **Long-running query detection + termination** - Query timeout enforcement
4. **Idle transaction detection** - Connection leak prevention
5. **Update ADR documentation** - Catch up on architectural decisions

## Sprint Backlog

| Issue | Task | Priority | Effort | Status |
|-------|------|----------|--------|--------|
| #119 | pg_stat_statements Graceful Degradation |
| #120 | Table Bloat Detector + VACUUM Action |
| #130 | Long-Running Query Detector + Termination |
| #131 | Idle Transaction Detector |
| - | Unit Tests (Table Bloat, VACUUM) |
| - | Test Container (test-table-bloat) |


## Success Criteria

### Must Have
- [ ] pg_stat_statements auto-enables or gracefully degrades
- [ ] Dashboard shows warning when features unavailable
- [ ] Table bloat detector triggers on >10% dead tuples
- [ ] VACUUM action clears dead tuples
- [ ] Long-running query detector triggers on queries > threshold
- [ ] Termination action kills long-running queries

### Should Have
- [ ] Idle transaction detector identifies abandoned connections
- [ ] Test containers for each new detector
- [ ] Unit test coverage for new detectors and actions

### Nice to Have
- [ ] Configurable thresholds via Dashboard for new detectors
- [ ] Dashboard visualisation of dead tuples over time

## Technical Approach

### Graceful Degradation (#119) 

- Check extension existence on connect
- Attempt CREATE EXTENSION if shared_preload_libraries configured
- Track availability in adapter state
- Expose via health endpoint
- Dashboard warning card in Settings

### Table Bloat (#120) 

- Query `pg_stat_user_tables` for `n_live_tup`, `n_dead_tup`
- Calculate bloat ratio
- Detector triggers at 10% threshold
- Action runs `VACUUM ANALYZE`
- Non-destructive, no rollback needed

### Long-Running Query (#130)

- Query `pg_stat_activity` for queries running > threshold
- Configurable threshold (default: 30 seconds)
- Action runs `pg_terminate_backend()` or `pg_cancel_backend()`
- Severity based on duration (30s = warning, 60s = critical)
- Safety: Don't terminate system queries or replication

### Idle Transaction (#131)

- Query `pg_stat_activity` for `idle in transaction` state
- Threshold: idle > 5 minutes in transaction
- Action: Terminate connection
- Prevents connection leaks from abandoned transactions

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Terminating wrong query | High | Filter by user, exclude system processes |
| VACUUM on large table blocks | Low | Standard VACUUM is non-blocking (not VACUUM FULL) |
| Idle transaction detection false positives | Medium | Conservative threshold (5 min), observe mode first |

## Definition of Done

- All code merged to `main` via PR
- CI/CD pipeline green
- Unit tests for new detectors and actions
- Test container demonstrates detection -> action flow
- ADRs updated for architectural decisions
- GitHub issues closed

## Notes

## Stretch Goals (If Ahead of Schedule)

- Configurable thresholds for all detectors via Dashboard