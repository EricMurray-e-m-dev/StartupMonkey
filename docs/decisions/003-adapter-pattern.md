# Decision 001: Using Adapter Pattern Database Abstraction

**Date:** 2025-10-13
**Status:** Accepted

## Context
StartupMonkey needs to support multiple different databases without rewriting core collection logic for each.
- Following SOLID principles here will lead to a robust design

## Decision
Use the **Adapter Pattern** with a `MetricInterface` to abstract database specific metrics.

## Consequences
- Each adapter handles one database only
- System is open for extension closed for modification
- Adapters are interchangable

## Alternatives Considered
- Stick all logic into one big file

