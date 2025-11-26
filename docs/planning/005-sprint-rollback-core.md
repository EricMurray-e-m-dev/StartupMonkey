# Sprint 005: Rollback + Core Actions

**Sprint:** 005  
**Dates:** Nov 27 - Dec 10 (2 weeks)  
**Theme:** Complete rollback implementation, add PgBouncer action, fix testing infrastructure

## Primary Objective

Make the system **production-ready for Christmas demo** by implementing rollback capability, adding connection pool management action, and ensuring reliable deployment via Docker.

## Objectives

1. **Implement + test rollback flow** (Dashboard UI + API + E2E test)
2. **Deploy PgBouncer action** (autonomous connection pool management)
3. **Fix integration tests** (update for Knowledge layer + delta tracking)
4. **Fix Docker deployment** (working `docker-compose up` with all services)

## Sprint Backlog

| Issue | Task | Priority | Effort |
|-------|------|----------|--------|
| #76 | Implement Rollback API Endpoint [Executor] | High | 2h |
| #77 | Add Rollback Button to Dashboard | High | 2h |
| #78 | Test Create Index Rollback E2E | High | 3h |
| #79 | Deploy PgBouncer Action [Executor] | High | 6h |
| #80 | Test PgBouncer Deployment E2E | High | 4h |
| #86 | Fix Docker Integration Tests | High | 6h |
| #87 | Create Docker Release Build (alpha-0.0.1) | High | 4h |
| #82 | Add System Metrics Collection [Collector] | Medium | 3h |

**Total Estimated Effort:** 30 hours over 2 weeks (~15 hours/week)

## Success Criteria

### Must Have
- [ ] Rollback button working in Dashboard for `create_index` action
- [ ] E2E test proves rollback drops index from database
- [ ] PgBouncer deploys automatically on connection pool exhaustion
- [ ] Integration tests passing in CI/CD pipeline
- [ ] `docker-compose up` starts all services with health checks green

### Should Have
- [ ] System metrics (CPU, memory) collected alongside DB metrics
- [ ] PgBouncer rollback tested (container removed)
- [ ] Docker release tagged as `alpha-0.0.1`

### Nice to Have
- [ ] Connection pool detector uses system metrics for accuracy
- [ ] Action history visible in Dashboard

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| PgBouncer connection string rewrite is complex | High | Use simple Docker network routing first, app-level config later |
| Integration tests might uncover architectural issues | Medium | Fix tests incrementally, prioritise smoke tests over comprehensive coverage |
| Docker Compose with 5+ services might be fragile | Medium | Add health check dependencies, use `restart: unless-stopped` |
| System metrics collection might be noisy | Low | Start with basic CPU/memory, skip disk I/O for MVP |

## Sprint Schedule

### Week 1 (Nov 27 - Dec 3)
**Focus:** Rollback + PgBouncer

- **Day 1-2:** Issues #76-78 (Rollback implementation + testing)
- **Day 3-4:** Issue #79 (PgBouncer action)
- **Day 5:** Issue #80 (PgBouncer E2E test)

### Week 2 (Dec 4 - Dec 10)
**Focus:** Testing + Docker

- **Day 6-7:** Issue #86 (Fix integration tests)
- **Day 8:** Issue #87 (Docker release build)
- **Day 9:** Issue #82 (System metrics - optional)
- **Day 10:** Buffer day for blockers + retrospective

## Definition of Done

- All code merged to `main` via PR
- CI/CD pipeline green (lint + unit tests + integration tests passing)
- Docker Compose starts all services successfully
- Manual testing confirms:
  - Rollback drops created index
  - PgBouncer container deploys and handles connections
  - Dashboard displays actions and allows rollback
- Sprint retrospective completed
- GitHub issues closed/moved appropriately
- Updated architecture documentation (if needed)

## Notes

- **PgBouncer connection rewrite:** Initially deploy PgBouncer in Docker network, have dummy app connect to `pgbouncer:6432` instead of `postgres:5432`. Don't try to dynamically rewrite connection strings in production apps yet - that's future work.

- **Integration test strategy:** Focus on smoke tests first (service starts → health check passes → basic flow works). Don't aim for 100% coverage, aim for confidence that core flows work.

- **Alpha release scope:** Just get it deployable. Don't worry about production-readiness, security hardening, or performance tuning yet. Goal is Christmas demo, not production launch.

- **System metrics:** If this takes longer than 3 hours, postpone to Sprint 6. It's a "nice to have" for improving detection accuracy, not blocking for demo.

## Stretch Goals (If Ahead of Schedule)

- #81: Tune PostgreSQL Cache Action
- #85: Dashboard Action History Page
- #83: Improve Connection Pool Detector with system metrics