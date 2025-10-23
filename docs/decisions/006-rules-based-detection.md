# Decision 008: Rules-Based Detection Engine

**Date:** 2025-10-23
**Status:** Accepted

## Context

StartupMonkey needs to detect database performance issues from normalized metrics.
- How to support multiple database types (Postgres, MySQL, MongoDB, SQLite)?
- How to make the system extensible for future detectors (ML-based, new databases etc.)?
- How to ensure detectors remain testable and maintainable?

## Decision

Implement a rules-based detection engine using SOLID principles:
- **Detector Interface:** All detectors implement a common interface with `Detect()`, `Name()`, and `Category()` methods
- **Engine Pattern:** Registry-based engine that runs all registered detectors on each metric snapshot
- **Domain Model:** Detectors work with `NormalisedMetrics` (domain model), not protobuf directly
- **Database-Agnostic:** Each detector provides database-specific recommendations via switch statements
- **Severity Levels:** Dynamic severity (info/warning/critical) based on threshold multiples

Built 4 initial detectors:
1. Missing Index Detector (sequential scans)
2. Connection Pool Detector (connection exhaustion)
3. Cache Miss Detector (low cache hit rate)
4. High Latency Detector (slow query execution)

## Consequences

**Positive:**
- New detectors added by implementing interface (Open/Closed Principle)
- Detectors decoupled from transport layer (gRPC/protobuf changes don't break detectors)
- Easy to test (create `NormalisedMetrics` structs without protobuf machinery)
- Database-agnostic recommendations guide users to correct tools (PgBouncer vs ProxySQL)
- Clear separation: Collector normalizes, Analyser detects, Executor acts

**Negative:**
- Conversion layer needed (`toNormalisedMetrics()`) in gRPC server
- Database-specific recommendations require maintenance when adding new databases
- Rule-based thresholds may produce false positives

**Trade-offs Accepted:**
- Slight conversion overhead for architectural cleanliness
- Manual threshold tuning vs automatic learning

## Alternatives Considered

**Detectors Use Protobuf Directly:**
- Rejected: Couples business logic to transport layer, harder to test, violates Clean Architecture

**Single Monolithic Detector:**
- Rejected: Violates Single Responsibility, harder to maintain, less extensible

**Threshold Configuration Files:**
- Deferred: Hardcoded thresholds sufficient for dissertation scope, can externalise later if needed