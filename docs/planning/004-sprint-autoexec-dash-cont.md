# Sprint 004: Autonomous Execution + Dashboard

## Primary Objective
Continue on issues from sprint 003. Some major hurdles taken out of the way should be able to flush out the rest of the issues relatively quickly.

## Objectives
1. Create more action deployments (Deploy PgBouncer, Redis Container)
2. Build out Dashboard more, display live detections(Analyser) & Action Queue(Executor)
3. E2E New test, new full E2E with deployed action

| Issue | Task |
|-------|------|
| #46 | Action queue visualisation |
| #47 | Implement Create Index action |
| #48 | Add Redis Container for caching |
| #49 | Deploy PgBouncer action |
| #50 | Action results publishing |
| #51 | E2E Test |
| #52 | Dashboard Polish + Styling  |


## Success Criteria
- [ ] Executor deploying caching/PgBouncer for connections
- [ ] Dashboard displaying live data, metrics, actions, etc.
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