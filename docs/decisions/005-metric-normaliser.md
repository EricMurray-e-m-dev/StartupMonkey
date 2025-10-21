# Decision 005: Metric Normaliser

**Date:** 2025-10-21
**Status:** Accepted

## Context
Due to database metrics not being uniform across different DB systems. We need to normalise the raw data pulled.
- Pull the raw data from source
- Normalise data into structure
- Normalising data also calculating scores for "health"

## Decision
Write a normaliser for each DB to organise the data into our structure.

## Consequences
- All data hitting the Analyser will be the same
- More abstraction in code
- Each DB now needs an Adapter (to connect to DB) AND a normaliser (to format the DB specific data)

## Alternatives Considered
- Write seperata gRPC contracts for each database - Too much technical debt to change in the future as support grows
