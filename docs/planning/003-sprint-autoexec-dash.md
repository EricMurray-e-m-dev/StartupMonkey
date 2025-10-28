# Sprint 003: Autonomous Execution + Dashboard

## Primary Objective
Create action deployments and execute automatically. Start building out Dashboard for UX Display

## Objectives
1. Create action deployments (Create Index Concurrently, Deploy PgBouncer etc.)
2. Build out Dashboard to show some real time metrics
3. E2E New test, new full E2E with deployed action

| Issue | Task |
|-------|------|
| #43 | Dashboard Foundation |
| #44 | Real-time metrics display |
| #45 | Show live detections |
| #46 | Action queue visualisation |
| #47 | Implement Create Index action |
| #48 | Add Redis Container for caching |
| #49 | Deploy PgBouncer action |
| #50 | Action results publishing |
| #51 | E2E Test |
| #52 | Dashboard Polish + Styling  |


## Success Criteria
- [ ] Dashboard setup & accessible
- [ ] Executor, deploying Create Index Concurrently
- [ ] Executor deploying caching/PgBouncer for connections
- [ ] Dashvoard displaying live data, metrics, actions, etc.
- [ ] E2E Integration Test passing



## Risks
- Testing is becoming more important as we go. Simulating true to life traffic etc is a challenge.


## Definition of Done
- All code merged to `main` via PR
- CI/CD pipeline green (lint + test pass + integration)
- Sprint retrospective done
- Github issues appropriated closed/moved & managed


## Notes
- None