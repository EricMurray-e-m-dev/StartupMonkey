# Sprint Retrospective

**Sprint:** 005
**Dates:** Nov 27 - Jan 9
**Theme:** Rollback + Core Actions

## What Went Well

- Rollback implementation completed - Dashboard button working, API endpoint functional, index drops confirmed via E2E test
- PgBouncer action deployed - Container spins up on connection pool exhaustion, routing traffic through connection pooler
- Docker Compose finally works - All services start with health checks, `docker-compose up` is reliable for local development
- Integration tests fixed - Updated for Knowledge layer + delta tracking, CI/CD pipeline green again
- Alpha release tagged - `alpha-0.1.0` available for Christmas demo

## What Didn't Go Well

- System metrics collection postponed - Ran out of time, pushed to Sprint 6
- Action history page not started - Stretch goal never reached
- Connection pool detector still using basic thresholds - Wanted to integrate system metrics but couldn't

## What We Learned

- Docker health check dependencies are essential - Services starting in wrong order caused hours of debugging
- PgBouncer connection routing simpler than expected - Docker network routing avoided connection string rewriting
- Integration test maintenance is ongoing work - Tests break with every architectural change, need to account for this in sprint planning

## What We'll Change Next Sprint

- Budget time for test maintenance - Every feature needs integration test updates, not just unit tests
- Document Docker setup - Compose file is complex, needs README updates for new contributors
- Address PgBouncer rollback - Can't leave untested code paths, fix this early
- Add system metrics properly - Detector accuracy suffers without CPU/memory context

## Completed Issues

- #76: Rollback API Endpoint
- #77: Rollback Button Dashboard
- #78: Create Index Rollback E2E
- #79: PgBouncer Action
- #80: PgBouncer Deployment E2E
- #86: Integration Tests Fixed
- #87: Docker Release Build

## Postponed Issues

- #82: System Metrics Collection (moved to Sprint 6)
- #81: Tune PostgreSQL Cache Action (stretch goal, not started)
- #85: Dashboard Action History Page (stretch goal, not started)
- #83: Connection Pool Detector improvements (blocked by #82)
