# Decision 009: Separate Database Adapters per Service

**Date:** 2025-11-07  
**Status:** Accepted

## Context

Both Collector and Executor need to interact with the target database, raising the question: should they share a common database adapter package?

**Collector needs:**
- `CollectMetrics()` - Read-only queries to pg_stat_statements, pg_stat_user_tables
- `GetConnectionInfo()` - Connection details for registration
- Minimal permissions (SELECT on system tables)

**Executor needs:**
- `CreateIndex()` - DDL operations
- `DropIndex()` - DDL operations for rollback
- `IndexExists()` - Validation before/after actions
- `GetCapabilities()` - Check what operations the DB supports
- Elevated permissions (CREATE INDEX, ALTER TABLE)

## Decision

Implement **separate database adapters** for Collector and Executor. No shared adapter package.

**Collector Adapter Interface:**
```go
type DatabaseAdapter interface {
    CollectMetrics(ctx context.Context) (*Metrics, error)
    GetConnectionInfo() ConnectionInfo
    Close() error
}
```

**Executor Adapter Interface:**
```go
type DatabaseAdapter interface {
    CreateIndex(ctx context.Context, params IndexParams) error
    DropIndex(ctx context.Context, indexName string) error
    IndexExists(ctx context.Context, indexName string) (bool, error)
    GetCapabilities() Capabilities
    Close() error
}
```

## Consequences

**Positive:**
- Each adapter has exactly the methods its service needs (Interface Segregation Principle)
- Services can evolve independently (different release cycles)
- Clearer security model (Collector adapter never has write permissions)
- Simpler testing (mock only what each service uses)
- Follows microservices principle: services should be independent

**Negative:**
- Some code duplication (connection handling, config parsing)
- Two adapters to maintain when adding new database support
- Developers must understand both interfaces

**Trade-offs Accepted:**
- Code duplication is acceptable for architectural clarity
- Aligns with Go proverb: "A little copying is better than a little dependency"

## Alternatives Considered

**Shared Adapter Package:**
- Rejected: Would couple services together, one interface can't serve both read and write needs cleanly

**Single Interface with All Methods:**
- Rejected: Violates Interface Segregation, Collector would have unused write methods

**Inheritance/Embedding:**
- Rejected: Go doesn't have inheritance, embedding would still create coupling
