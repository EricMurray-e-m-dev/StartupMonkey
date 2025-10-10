# Sprint 01: gRPC Communication

## Primary Objective
Establish gRPC communication between Collector and Analyser.

## Objectives
1. Design database-agnostic gRPC contract
2. Implement Collector that sends metrics to Analyzer
3. Implement Analyzer that receives and logs metrics
4. Integration test proving end-to-end communication works

| Issue | Task |
|-------|------|
| #4 | Design database-agnostic metric schema |
| #5 | Implement protobuf contract |
| #6 | Collector PostgreSQL adapter (minimal) |
| #7 | Analyzer gRPC server (echo mode) |
| #8 | Integration test: Collector → Analyzer |


## Success Criteria
- [ ] Collector sends metrics to Analyzer via gRPC stream
- [ ] Analyzer receives and logs metrics (JSON format)
- [ ] Integration test passes in CI/CD
- [ ] Architecture diagrams complete and committed
- [ ] Contract supports future database types (not PostgreSQL-specific)


## Risks
- gRPC learning curve
- Contract not database agnostic (KEY)


## Definition of Done
- All code merged to `main` via PR
- CI/CD pipeline green (lint + test pass)
- Architecture diagrams in `~/docs/architecture/diagrams/`
- Sprint retrospective done
- Github issues appropriated closed/moved & managed


## Notes
First day working on proto contracts. Very crucial stage of the development. Need to try and make the contracts as DB agnostic as possible. Changing contracts in the future will add big technical debt. Aim is to have a loose contract that I can spin up and just create adapters for that "plug" straight into the setup. SOLID important here.
