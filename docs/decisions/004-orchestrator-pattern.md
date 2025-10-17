# Decision 004: Orchestrator Pattern for Collector

**Date:** 2025-10-16  
**Status:** Accepted


## Context

The Collector service has two independent components:
- **DatabaseAdapter** - Collects metrics from plugged in adapter database
- **gRPC Client** - Sends metrics to Analyser service

These components need to work together in a continuous loop: collect metrics → send to Analyser → repeat. However, they should remain decoupled - the adapter shouldn't know about gRPC, and the client shouldn't know about PostgreSQL.

## Decision

Implement an **Orchestrator** that coordinates the collection and transmission workflow without coupling the components together.

### Why Orchestrator Pattern?

**Definition:** A central component that controls workflow between multiple services/components without those components knowing about each other.

**Fits our use case because:**
- Central coordination of lifecycle (startup, loop, shutdown)
- One-way data flow (collect → send)
- Components remain decoupled
- Clear separation of concerns

### Alternatives Considered

**Mediator Pattern:**
- Implies bidirectional communication between components
- Our flow is unidirectional (collect → send)
- Unnecessarily complex for our needs

**Pipeline Pattern:**
- Implies data transformation at each stage
- We're moving data, not transforming it
- Orchestrator does more than pass data (manages lifecycle)

**Controller Pattern:**
- Implies request/response model (web framework context)
- We have continuous loop, not request-driven
- Wrong semantic fit