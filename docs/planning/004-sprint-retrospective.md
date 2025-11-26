# Sprint Retrospective

**Sprint:** 004
**Dates:** Nov 10 - Nov 23
**Completed Issues:** 

## What Went Well

- Knowledge Layer implemented - Built entire MAPE-K architecture with Redis-backed state management, arguably the most significant architectural addition to the project
- Detection deduplication solved - Delta tracking + Knowledge layer eliminated duplicate detections/actions, making the system truly autonomous
- Dashboard state management working - Node script background process persists NATS data across page navigation, smooth UX without loading screens
- Create Index action fully working - First autonomous action confirmed working in manual tests with proper validation + rollback stub
- Delta-based detection is brilliant - Moving from absolute thresholds to rate-of-change prevents false positives and makes detections much more intelligent
- Multi-service integration solid - Collector → Analyser → Executor → Knowledge all communicating properly via gRPC + NATS

## What Didn't Go Well

- Scope exploded mid-sprint - Started with 7 issues (#46-52), ended up implementing 13+ issues due to architectural pivots
- Integration tests fell behind - Focused on getting architecture right, tests are now broken/outdated
- Docker deployment hasn't been touched - CI/CD issues not addressed, still manual testing only
- Rollback never tested - Code exists but never actually triggered or validated in practice
- PgBouncer/Redis actions postponed - Got blocked by Knowledge layer work, didn't make it to these issues

## What We Learned

- MAPE-K was the right call - Autonomous systems need feedback loops, Knowledge layer provides this
- Delta tracking > absolute thresholds - Much more sophisticated detection strategy, prevents alert fatigue
- Go is fast to build with - Entire Knowledge service + integrations built in ~3 days, validates tech stack choice
- Background processes in Next.js are painful - Had to hack around framework limitations with external Node script
- Single database assumption is limiting - Hardcoded connection string blocks multi-DB support, need to address

## What We'll Change Next Sprint

- Fix integration tests FIRST - Can't keep building features without reliable test coverage
- One architectural change at a time - Knowledge layer was necessary but derailed 2 weeks, no more surprise refactors
- Test rollback immediately - It's implemented but untested, that's technical debt
- Stick to the sprint plan - If new issues arise, put them in backlog for next sprint unless they're blocking
- Docker must work - Can't do a Christmas demo without reliable deployment