# Sprint 006: Autonomous Rollback

**Sprint:** 006  
**Dates:** Jan 12 - Jan 23 (2 weeks)  
**Theme:** Autonomous rollback on performance degradation

## Primary Objective

Implement **autonomous rollback** so the system can detect when an action causes performance degradation and automatically revert it without human intervention.

## Objectives

1. **Implement autonomous rollback** (detect degradation post-action, trigger rollback automatically)
2. **Add system metrics collection** (carried over from Sprint 005)
3. **Build action history page** (visibility into what the system has done)
4. **Improve connection pool detector** (use system metrics for better accuracy)

## Sprint Backlog

| Issue | Task | Priority | Effort |
|-------|------|----------|--------|
| #95 | Autonomous Rollback on Degradation [Executor] | High | 8h |
| #86 | Add System Metrics Collection [Collector] | High | 4h |
| #88 | Dashboard Action History Page [Dashboard] | Medium | 4h |
| #87 | Improve Connection Pool Detector [Analyser] | Medium | 3h |
| #48 | Tune PostgreSQL Cache Action | Low | 3h |

**Total Estimated Effort:** 22 hours over 2 weeks (~11 hours/week)

## Success Criteria

### Must Have
- [ ] Autonomous rollback triggers when action causes >10% performance degradation
- [ ] Rollback logged in Knowledge layer with reason
- [ ] E2E test proves: action executes -> metrics degrade -> rollback triggered automatically
- [ ] System metrics (CPU, memory) collected alongside DB metrics

### Should Have
- [ ] Action history page shows all executed actions with status (applied/rolled back)
- [ ] Connection pool detector uses CPU/memory alongside connection count
- [ ] PostgreSQL cache action tuned for real workloads

### Nice to Have
- [ ] Configurable degradation threshold (not hardcoded 10%)
- [ ] Rollback notification via NATS event for Dashboard

## Technical Approach

### Autonomous Rollback (#95)

1. After action execution, Executor enters "observation window" (30-60 seconds)
2. Collector continues sending metrics to Analyser
3. Analyser compares post-action metrics against pre-action baseline
4. If degradation detected (query latency up, throughput down), Analyser publishes `action.degraded` event
5. Executor receives event, checks if rollback is available, executes rollback
6. Knowledge layer updated with rollback record + reason

Key decision: Observation window length. Too short = false positives. Too long = slow response.

### System Metrics (#86)

- Collect via `/proc` on Linux or `gopsutil` library
- Metrics: CPU usage %, memory usage %, load average
- Publish alongside DB metrics on same interval
- Store in Knowledge layer for baseline comparisons

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Rollback triggers on normal variance | High | Use delta tracking, require sustained degradation over multiple samples |
| Observation window too short | Medium | Start with 60 seconds, make configurable |
| System metrics collection adds latency | Low | Collect async, don't block DB metric collection |
| Action history page scope creep | Low | MVP: table with action name, timestamp, status. No filtering/search yet |

## Sprint Schedule

### Week 1 (Jan 12 - Jan 16)
**Focus:** Autonomous rollback core

- **Day 1-2:** #95 - Implement observation window + degradation detection
- **Day 3-4:** #95 - Wire up rollback trigger + Knowledge layer updates
- **Day 5:** #95 - E2E test for autonomous rollback

### Week 2 (Jan 19 - Jan 23)
**Focus:** System metrics + Dashboard

- **Day 6:** #86 - System metrics collection
- **Day 7:** #87 - Improve connection pool detector with system metrics
- **Day 8-9:** #88 - Action history page
- **Day 10:** #48 - PostgreSQL cache tuning + buffer

## Definition of Done

- All code merged to `main` via PR
- CI/CD pipeline green
- E2E test confirms autonomous rollback flow
- Manual testing confirms:
  - Create index -> latency spike -> automatic rollback
  - Action history shows rollback with reason
  - System metrics visible in Collector logs
- Sprint retrospective completed
- GitHub issues closed

## Dependencies

- #86 (System Metrics) should complete before #87 (Connection Pool Detector improvement)
- #95 (Autonomous Rollback) is independent, can proceed in parallel

## Notes

- **Degradation threshold:** Start with 10% increase in p95 latency or 10% decrease in throughput. This is arbitrary but gives us something to tune.

- **Observation window:** 60 seconds with samples every 5 seconds = 12 data points. Require 3+ consecutive degraded samples before triggering rollback.

- **Action history MVP:** Just a table. No pagination, no filtering, no export. Get data visible first, polish later.

- **PostgreSQL cache tuning:** Low priority. Only attempt if ahead of schedule. This has been on the backlog since Sprint 4.

## Stretch Goals (If Ahead of Schedule)

- #96: Action Approval Mode (semi-autonomous, human confirms before execute)
- #97: Configuration UI (adjust thresholds from Dashboard)
