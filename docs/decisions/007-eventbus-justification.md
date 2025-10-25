# Decision 007: Event Bus for Service Decoupling

**Date:** 2025-10-25  
**Status:** Accepted

## Context

StartupMonkey's Analyser service detects database performance issues and needs to communicate these detections to the Executor service for action. We evaluated several communication patterns:

**Requirements:**
- Analyser should not be tightly coupled to Executor (services should be independently deployable)
- Detections are asynchronous (fire-and-forget, no response needed from Executor)
- System should be extensible (future services like Dashboard, notification service, audit log should receive detections)
- Minimal latency overhead acceptable (detection delivery within seconds is sufficient)
- Lightweight infrastructure (aligns with StartupMonkey's minimal footprint goal)

**Existing Communication:**
- Collector → Analyser uses gRPC (synchronous streaming makes sense for metric collection)

## Decision

Use **NATS** as an event bus for Analyser → Executor communication.

**Architecture:**
```
Collector → gRPC → Analyser → NATS (pub) → Executor (sub)
                              ↓
                         Future: Dashboard, Notifications, Audit Log
```

**Communication Patterns:**
- **Collector → Analyser:** gRPC streaming (synchronous, bidirectional)
- **Analyser → Executor:** NATS pub/sub (asynchronous, decoupled)
- **Dashboard → Executor:** gRPC (synchronous, request/response for queries/approvals - Sprint 4)

## Rationale

### Why Event Bus Over Direct gRPC?

**gRPC Issues for This Use Case:**
1. **Tight Coupling:** Analyser must know Executor's address and manage connection
2. **Single Consumer:** Hard to add new subscribers (Dashboard, notifications, etc.)
3. **No Message Buffering:** If Executor is down, detections are lost
4. **Blocking:** Analyser would need to wait for Executor acknowledgment

**Event Bus Benefits:**
1. **Decoupling:** Analyser publishes to topic, doesn't know who consumes
2. **Multiple Consumers:** Dashboard, notifications, audit log can subscribe to same detections
3. **Message Persistence:** NATS JetStream buffers messages if Executor is down
4. **Non-Blocking:** Analyser publishes and continues immediately
5. **Scalability:** Multiple Executor instances can subscribe (future load balancing)

### Why NATS Over RabbitMQ?

| Feature | NATS | RabbitMQ | Winner |
|---------|------|----------|--------|
| Setup Complexity | Simple (single binary) | Medium (Erlang runtime) | NATS |
| Docker Image Size | ~15MB | ~200MB | NATS |
| Memory Footprint | ~10-20MB | ~100MB+ | NATS |
| Startup Time | <1s | ~5s | NATS |
| Message Persistence | Optional (JetStream) | Built-in | RabbitMQ |
| Ordering Guarantees | Best-effort | Strict | RabbitMQ |
| Golang Integration | Native | Good | NATS |

**Decision: NATS**
- Aligns with StartupMonkey's lightweight philosophy
- Sufficient guarantees for our use case (detection delivery is best-effort, not critical)
- Simpler to operate and debug
- Faster startup in Docker Compose (better developer experience)

### Message Persistence Trade-off

**NATS JetStream (persistent):**
- Messages buffered if Executor down
- Guarantees delivery when Executor restarts
- Slight complexity increase

**NATS Core (in-memory):**
- Fire-and-forget (lost if Executor down)
- Simpler, faster
- Acceptable for our use case (detections repeat every 30s anyway)

**Decision:** Start with NATS Core, 

## Consequences

**Positive:**
- Services fully decoupled (Analyser/Executor independently deployable)
- Easy to add new subscribers (Dashboard live updates, notification service)
- No blocking in Analyser (publishes and moves on)
- Scalable architecture (multiple Executor instances possible)
- Small infrastructure footprint (~15MB Docker image, ~10MB RAM)

**Negative:**
- Additional infrastructure component to monitor (NATS)
- No synchronous feedback to Analyser (can't confirm Executor received detection)
- Requires JSON serialization overhead (vs protobuf in gRPC)
- Eventual consistency (slight delay between publish and consumption)

**Acceptable Trade-offs:**
- No feedback needed: Analyser just publishes detections, doesn't care about execution outcome
- JSON overhead negligible: Detection messages are small (~1-2KB)
- Eventual consistency fine: Detection delivery within seconds is sufficient (not milliseconds)

## Alternatives Considered

**Direct gRPC (Analyser → Executor):**
- Rejected: Tight coupling, single consumer, blocking

**Database as Queue:**
- Rejected: Adds database dependency, polling overhead, not real-time

**HTTP Webhooks:**
- Rejected: Requires Executor to expose HTTP endpoint, retry logic complex, no multi-consumer

**Redis Pub/Sub:**
- Rejected: Similar to NATS but heavier, requires Redis running

**Kafka:**
- Rejected: Massive overkill (high throughput not needed), heavy infrastructure