# Sprint 008: Idle Transactions + System Metrics

**Sprint:** 008
**Dates:** Feb 9 - Feb 20 (2 weeks)
**Theme:** Complete detector suite and expand observability

## Primary Objective

Finish the **Idle Transaction Detector** and finally tackle **System Metrics Collection** which has been postponed since Sprint 005.

## Objectives

1. **Idle transaction detection + termination** - Carried from Sprint 007
2. **System metrics collection** - CPU, memory, load average
3. **Webhooks** - External notifications for detections/actions

## Sprint Backlog

| Issue | Task |
|-------|------|
| #131 | Idle Transaction Detector |
| - | Test Container (test-idle-transaction) |
| - | Unit Tests (Idle Transaction) |
| #86 | System Metrics Collection |
| #118 | Webhooks |


## Success Criteria

### Must Have
- [ ] Idle transaction detector triggers on connections idle in transaction > 5 minutes
- [ ] Termination action closes idle transaction connections
- [ ] Test container demonstrates idle transaction detection
- [ ] Unit tests for detector

### Should Have
- [ ] System metrics (CPU, memory, load) collected alongside database metrics
- [ ] System metrics visible in Dashboard

### Nice to Have
- [ ] Webhooks for detection/action notifications
- [ ] Webhook configuration in Dashboard

## Technical Approach

### Idle Transaction (#131)

Reuses existing infrastructure:
- **Collector**: Query `pg_stat_activity` for `state = 'idle in transaction'`
- **Analyser**: New `IdleTransactionDetector`, threshold 5 minutes
- **Executor**: Reuse `TerminateQueryAction` - same PID termination
```sql
SELECT pid, usename, state, query,
       EXTRACT(EPOCH FROM (now() - state_change)) as idle_duration_secs
FROM pg_stat_activity
WHERE state = 'idle in transaction'
AND state_change < now() - interval '5 minutes'
AND pid != pg_backend_pid()
```

### System Metrics (#86)

- Use `gopsutil` library for cross-platform metrics
- Collect: CPU usage %, memory usage %, load average
- Add to RawMetrics alongside database metrics
- No new detector initially - data collection first
- Skip if not availble -> Remote connection to a DB

### Webhooks (#118)

- POST to configured URL on detection or action completion
- Configurable in Dashboard settings
- Payload: detection/action JSON
- Optional: Slack/Discord formatting

## Backlog (Future Sprints)

- #119: Multi-DB Monitoring
- #93: MySQL Adapter
- #94: MongoDB Adapter

## Definition of Done

- All code merged to `main` via PR
- CI/CD pipeline green
- Unit tests for idle transaction detector
- Test container demonstrates detection -> termination flow
- GitHub issues closed

## Notes

- System metrics (#86) has been postponed four sprints - getting it done this time
- Webhooks are nice-to-have but add user value for notifications