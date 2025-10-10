# Decision 002: Design Decisions for Proto Contracts for gRPC

**Date:** 2025-10-10
**Status:** Accepted

## Context
To generate our gRPC code, we need to define solid contracts. Both SOLID and solid.
- Needs to be Database agnostic - Contract shouldnt care what DB is where
- Need to incorporate something to allow it to be flexible, different DBs have different logs/stats
- How we handle the streaming also, Bidirectional is the most scalable but probably overkill for right now

## Decision
To keep our contracts loose, we will use maps to just map in data, we dont care what the data is. The services themselves will decide how to handle what based off of our adapters. For now we will just implement client streaming, not bidirectional.

## Consequences
- Our contracts are very loose so we have to be really implicit with out adapters which isn't a bad thing
- Client streaming is more efficient than unary streaming but less than Bidirectional, fine for now
- Possible techincal debt here needing to upgrade to bidirectional later

## Alternatives Considered
- Database specific fields in contracts, hybrid approach - Too prone to change, large techincal debt
- Bidirectional streaming, both services talking to each other, more complex to implement overkill for now, will add later
