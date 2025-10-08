# Decision 001: Go for Core Services Instead of Python

**Date:** 2025-10-08
**Status:** Accepted

## Context
Original plan used Python for Analyser/Executor for quick iteration & large ecosystem. Concerned about:
- Image size (Python: 300MB+ vs Go: <50MB)
- Performance for production workloads
- Employability (Go microservices more in-demand + more impressive)

## Decision
Use Go for Collector, Analyser, and Executor. Python only for optional ML service or other services (Future Work/Stretch Goals).

## Consequences
- Smaller footprint (aligns with project goals, minimal footprint)
- Single language for core (less context switching)
- ONNX deployment in Go (learning curve)
- Multi-language still demonstrated if more services added outside of Core 4

## Alternatives Considered
- Pure Python stack (rejected: footprint too large, unimpressive & Go far more performant)
- Pure Go with Go ML libs (rejected: Python ML ecosystem superior)