# Decision 011: Separate NATS Worker for Dashboard

**Date:** 2025-11-11  
**Status:** Accepted

## Context

The Dashboard needs real-time updates from NATS for:
- Live metrics from Collector
- New detections from Analyser
- Action status updates from Executor

**Problem:** Next.js is designed for request/response HTTP patterns. It does not support long-running background processes or persistent connections within the application lifecycle.

**Attempts that failed:**
1. NATS subscription in API route - Connection closed after request completed
2. NATS subscription in middleware - Same issue, no persistent process
3. Custom server.js - Conflicts with Next.js build/deployment model

## Decision

Run a **separate Node.js worker script** alongside the Next.js application that:
1. Maintains persistent NATS subscriptions
2. Receives events and stores them (or forwards to Dashboard via internal mechanism)

**Architecture:**
```
NATS ──► nats-worker.js ──► Shared State/API ──► Next.js Dashboard
```

**Docker Compose Configuration:**
```yaml
dashboard:
  build: ./dashboard
  command: sh -c "node nats-worker.js & npm start"
```

The worker runs as a background process in the same container, subscribing to:
- `metrics.*` - For live metrics display
- `detections.*` - For detection feed
- `actions.*` - For action status updates

## Consequences

**Positive:**
- Clean separation between HTTP serving (Next.js) and event consumption (worker)
- Worker can maintain persistent NATS connection
- Dashboard receives real-time updates
- No fighting against Next.js architecture

**Negative:**
- Two processes in one container (slightly against container best practices)
- Need to coordinate state between worker and Next.js
- Additional complexity in startup/health checks
- Worker crash doesn't automatically restart (mitigated by process manager)

**Trade-offs Accepted:**
- Two processes in one container is pragmatic for this scale
- Could split into separate containers later if needed

## Alternatives Considered

**WebSocket from Browser Direct to NATS:**
- Rejected: NATS doesn't expose WebSocket by default, would need NATS WebSocket gateway

**Polling API:**
- Rejected: Not real-time, unnecessary load on Knowledge service

**Server-Sent Events (SSE) from Next.js:**
- Rejected: Still need something to receive NATS events server-side

**Separate Worker Container:**
- Considered but deferred: Adds Docker Compose complexity, one container is simpler for now

**Different Frontend Framework:**
- Rejected: Next.js/React ecosystem benefits outweigh this limitation
