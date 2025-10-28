# Sprint 002: Detection & Action Pipeline

## Primary Objective
Build out rules based issue detection, implement action pipeline to queue actions

## Objectives
1. Create detection engine in Analyser
2. Build out executor skeleton
3. Build dummy app for proper mock tests
4. E2E Integration Test to prove functionality

| Issue | Task |
|-------|------|
| #23 | Detection Engine |
| #24 | Expand/Rework Metrics |
| #25 | Create dummy app for mock |
| #26 | Executor Skeleton |
| #27 | Integrate Event Bus |
| #28 | E2E Integration Test |
| #29 | Health check endpoints |
| #30 | Docs Catchup  |


## Success Criteria
- [ ] Detection engine detects basic issues
- [ ] Analyser send recommended action to executor
- [ ] Event bus up and running
- [ ] Metrics properly reworked for future databases
- [ ] E2E Integration Test passing
- [ ] /health endpoints working



## Risks
- Metric rework could hide alot of technical debt
- Potential scope creep for one sprint


## Definition of Done
- All code merged to `main` via PR
- CI/CD pipeline green (lint + test pass + integration)
- Sprint retrospective done
- Github issues appropriated closed/moved & managed


## Notes
- None