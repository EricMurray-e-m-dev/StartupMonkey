# Decision 015: Graceful Degradation for Optional Extensions

**Date:** 2025-02-05  
**Status:** Accepted

## Context

StartupMonkey relies on `pg_stat_statements` for query-level analysis (slow query detection, index recommendations). However:

1. `pg_stat_statements` requires `shared_preload_libraries` config - needs PostgreSQL restart
2. Cloud databases (Supabase, RDS) usually have it pre-enabled
3. Local/self-hosted setups often don't
4. Failing to connect because of a missing optional extension is poor UX

**Target user consideration:**
"Vibe Coders" won't know how to configure PostgreSQL extensions. The system should work with reduced functionality rather than fail entirely.

## Decision

Implement **graceful degradation** for optional PostgreSQL extensions:

**On Connect:**
1. Check if extension exists via `pg_extension`
2. Check if extension is in `shared_preload_libraries`
3. Attempt `CREATE EXTENSION IF NOT EXISTS` if preloaded
4. Track availability in adapter state

**Availability Tracking:**
```go
type PostgresAdapter struct {
    pgStatStatementsAvailable bool
    // ...
}

func (p *PostgresAdapter) GetUnavailableFeatures() []string
```

**Health Endpoint:**
```json
{
  "status": "healthy",
  "unavailable_features": ["pg_stat_statements"]
}
```

**Dashboard Warning:**
Settings page shows warning card when features are unavailable, with instructions to enable.

**Feature Guards:**
```go
func (p *PostgresAdapter) analyseSlowQueries(ctx context.Context, tableName string) ([]string, error) {
    if !p.pgStatStatementsAvailable {
        return nil, fmt.Errorf("pg_stat_statements not available")
    }
    // ...
}
```

## Consequences

**Positive:**
- System works out-of-box on any PostgreSQL
- Clear feedback on what's missing and how to fix
- Full functionality when extensions available
- No startup failures for optional features
- Database-agnostic pattern (other DBs can report their unavailable features)

**Negative:**
- Some detectors have reduced accuracy without pg_stat_statements
- User might not notice the warning and wonder why index recommendations are missing

**Trade-offs Accepted:**
- Partial functionality is better than no functionality
- User education via Dashboard warnings

## Alternatives Considered

**Fail Fast:**
- Rejected: Poor UX for target users, blocks adoption

**Automatic postgresql.conf Modification:**
- Rejected: Requires file system access, restart, too invasive

**Require Extension as Prerequisite:**
- Rejected: Adds friction to onboarding, against "zero-config" goal