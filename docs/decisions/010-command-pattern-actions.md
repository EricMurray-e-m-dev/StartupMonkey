# Decision 010: Command Pattern for Executor Actions

**Date:** 2025-11-07  
**Status:** Accepted

## Context

The Executor service needs to apply various optimisation actions:
- Create database indexes
- Deploy PgBouncer container
- Deploy Redis container
- Tune PostgreSQL configuration

Key requirements:
1. **Rollback capability:** Every action must be reversible if it doesn't improve performance
2. **Validation:** Actions should verify preconditions before execution
3. **Extensibility:** Adding new action types should be straightforward
4. **Consistency:** All actions should follow the same lifecycle

## Decision

Implement the **Command Pattern** where each action is encapsulated as an object with standardised methods.

**Action Interface:**
```go
type Action interface {
    Execute(ctx context.Context) (*ActionResult, error)
    Rollback(ctx context.Context) error
    Validate(ctx context.Context) error
    GetMetadata() *ActionMetadata
}
```

**Implemented Actions:**
- `CreateIndexAction` - Creates index, rollback drops it
- `DeployPgBouncerAction` - Deploys container, rollback removes it
- `DeployRedisAction` - Deploys container, rollback removes it
- `UpdateCacheConfigAction` - Changes config, rollback restores original

**Execution Flow:**
1. Receive detection from NATS
2. Select appropriate Action based on detection type
3. Call `Validate()` - check preconditions (capabilities, existing state)
4. Call `Execute()` - apply the optimisation
5. Store result in Knowledge service
6. If improvement not measured → call `Rollback()`

## Consequences

**Positive:**
- Uniform interface for all actions (easy to add new types)
- Built-in rollback support (each action knows how to undo itself)
- Actions are self-contained (encapsulate all logic needed)
- Easy to test (each action testable in isolation)
- Metadata available for logging/dashboard display
- Supports action queuing and history tracking

**Negative:**
- More boilerplate than simple functions
- Each action needs both Execute and Rollback implemented
- State management required (action must remember what it did for rollback)

**Trade-offs Accepted:**
- Boilerplate is worthwhile for the architectural benefits
- Rollback complexity is essential for autonomous operation

## Alternatives Considered

**Simple Functions:**
```go
func CreateIndex(ctx, params) error
func RollbackCreateIndex(ctx, params) error
```
- Rejected: No encapsulation, rollback logic separated from execute, harder to track state

**Strategy Pattern Only:**
- Rejected: Strategy is for selecting algorithms, Command is for encapsulating operations with undo

**Event Sourcing:**
- Rejected: Overkill for our use case, adds significant complexity
