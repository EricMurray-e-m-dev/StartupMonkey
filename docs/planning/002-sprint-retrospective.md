# Sprint Retrospective

**Sprint:** 002
**Dates:** Oct 18 - Oct 24
**Completed Issues:** All Issues Closed

## What Went Well
- Setup proper detection rules based
- Load tested with Locust script , detections firing
- Wrote `Executor` skeleton and built out action queue
- Wrote full E2E integration test for data pipeline

## What Didn't Go Well
- Had to refactor contracts already added another day and a half of cleanup
- Go module imports are finiky with protected branches on Github

## What We'll Change Next Sprint
- Better flow for when and how to isolate changes
- If changing a service that will be relied on the future, must push to main seperately
- Allows go module imports to update to newest version `go mod tidy`
