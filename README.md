# StartupMonkey

Autonomous database performance optimisation system for early-stage startups and solo developers.

## The Problem

You've built an application using AI-assisted development. It works. Users arrive. Traffic grows. Then your database starts struggling - queries slow down, connections pile up, and you're staring at PostgreSQL logs with no idea what's wrong or how to fix it.

Hiring a DBA is expensive. Learning database internals takes months. Your users are experiencing slow load times right now.

## The Solution

StartupMonkey monitors your database, detects common performance issues, and fixes them automatically where possible. No complex configuration required. Connect it to your database and it starts working.

**What it detects:**
- Missing indexes causing slow queries
- Table bloat from dead tuples
- Long-running queries blocking resources
- Idle transactions holding locks
- Connection pool exhaustion
- Poor cache hit rates
- High latency

**What it does about it:**
- Creates indexes (non-blocking)
- Runs VACUUM to reclaim space
- Terminates problematic queries
- Deploys connection pooling (PgBouncer)
- Tunes database configuration
- Deploys Redis instance
- Rolls back changes if performance degrades

## Architecture

StartupMonkey implements the MAPE-K feedback loop as microservices.

| Service | Role | Tech |
|---------|------|------|
| Collector | Gathers metrics from your database | Go |
| Analyser | Detects performance issues using rules-based expert system | Go |
| Executor | Applies optimisations with rollback capability | Go |
| Knowledge | Stores configuration, state, and coordinates services | Go + Redis |
| Dashboard | Real-time visibility and control | Next.js + React |

**Communication:**
- gRPC for synchronous requests (Collector → Analyser)
- NATS for asynchronous events (Analyser → Executor, status updates)

## Features

### Detectors

| Detector | Trigger | Action |
|----------|---------|--------|
| Missing Index | High sequential scan rate | Create index (CONCURRENTLY) |
| Table Bloat | Dead tuples | VACUUM ANALYZE |
| Long-Running Query | Query duration | Terminate query |
| Idle Transaction | Idle in transaction | Terminate connection |
| Connection Pool | Usage | Deploy PgBouncer |
| Cache Miss | Hit rate | Configuration recommendations / Deploy Redis |
| High Latency | p95 Latency | Tune database configuration |

### Execution Modes

Control how much autonomy the system has:

- **Autonomous**: Detects and fixes issues automatically
- **Approval**: Detects issues, queues actions for your approval
- **Observe**: Detects and reports issues, takes no action

### Automatic Rollback

When an action is executed, StartupMonkey monitors subsequent metrics. If performance degrades, it automatically rolls back the change where possible. Some actions can be rolled back manually in the Dashboard by the user.

### Graceful Degradation

The system works with whatever PostgreSQL configuration you have. Optional extensions like `pg_stat_statements` are used when available, but the system continues functioning without them.

## Getting Started

### Prerequisites

- Docker and Docker Compose
- PostgreSQL 12+ database to monitor

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/EricMurray-e-m-dev/StartupMonkey.git
cd StartupMonkey
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Start all services:
```bash
docker-compose up --build
```

4. Open the Dashboard at `http://localhost:3000`

5. Add your database connection in Settings

### Service Ports

| Service | Port | Purpose |
|---------|------|---------|
| Dashboard | 3000 | Web UI |
| Collector | 8080 | Health check |
| Analyser | 8081 | Health check |
| Analyser | 50051 | gRPC |
| Executor | 8082 | Health check |
| Knowledge | 8083 | Health check |
| Knowledge | 50053 | gRPC |
| NATS | 4222 | Event bus |

## Configuration

### Environment Variables

**Database Thresholds:**

| Variable | Default | Description |
|----------|---------|-------------|
| `THRESHOLD_CONNECTION_POOL_CRITICAL` | 0.8 | Connection usage ratio to trigger alert |
| `THRESHOLD_SEQ_SCAN_DELTA` | 10.0 | Sequential scan increase to trigger |
| `THRESHOLD_P95_LATENCY_MS` | 500 | P95 latency threshold in milliseconds |
| `THRESHOLD_CACHE_HIT_RATE` | 0.9 | Minimum cache hit rate |
| `THRESHOLD_TABLE_BLOAT` | 0.1 | Dead tuple ratio (0.1 = 10%) |
| `THRESHOLD_LONG_QUERY_SECS` | 30 | Query duration to trigger termination |

**Service Configuration:**

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | nats://localhost:4222 | NATS server address |
| `KNOWLEDGE_ADDRESS` | localhost:50053 | Knowledge service gRPC address |
| `GRPC_PORT` | 50051 | Analyser gRPC port |

### Dashboard Settings

Thresholds and execution mode can be adjusted in the Dashboard Settings page. Changes are stored in the Knowledge service and applied without restart.

## How It Works

### Detection Flow

1. **Collector** polls your database every 10 seconds, gathering metrics from `pg_stat_activity`, `pg_stat_user_tables`, `pg_stat_statements`, and other system views.

2. **Normaliser** converts raw metrics into a standardised format with health scores and delta tracking.

3. **Analyser** runs all registered detectors against the normalised metrics. Each detector checks specific conditions and returns a detection if thresholds are exceeded.

4. **Detection** is published to NATS with action metadata (what to do, which table/query/connection).

5. **Executor** receives the detection, creates the appropriate action, and executes it based on the current execution mode.

6. **Action result** is stored in Knowledge and published to NATS for Dashboard updates.

### Rollback Flow

1. After action execution, the **Verification Tracker** monitors metrics for 3 collection cycles.

2. If metrics degrade beyond threshold, a rollback request is published.

3. **Executor** receives the rollback request and reverses the action (e.g., drops the created index).

4. Rollback result is logged and the detection is marked as requiring investigation.

## Project Structure
```
StartupMonkey/
├── collector/           # Metrics collection service
│   ├── cmd/
│   └── internal/
│       ├── adapter/     # Database adapters (PostgreSQL)
│       ├── normaliser/  # Metric normalisation
│       └── orchestrator/
├── analyser/            # Detection service
│   ├── cmd/
│   └── internal/
│       ├── detector/    # Detection algorithms
│       ├── engine/      # Detection orchestration
│       └── config/      # Threshold configuration
├── executor/            # Action execution service
│   ├── cmd/
│   └── internal/
│       ├── actions/     # Action implementations
│       ├── database/    # Database adapters
│       └── handler/     # Detection handling
├── knowledge/           # State management service
├── dashboard/           # Next.js web UI
├── proto/               # gRPC protocol definitions
├── docs/
│   ├── decisions/       # Architecture Decision Records
│   └── sprints/         # Sprint planning and retrospectives
└── tests/
    └── evaluation/      # Test containers for each detector
```

## Supported Databases

| Database | Status | Notes |
|----------|--------|-------|
| PostgreSQL | Supported | Full feature support |
| MySQL | Planned | Architecture supports it |
| MongoDB | Planned | Limited feature set |

## Development

### Running Tests
```bash
# Unit tests
cd service && go test ./tests/unit/... -v
```

### Test Containers

Purpose-built containers exist for testing each detector:
```bash
# Missing index detection
cd tests/evaluation/test-missing-index && docker-compose up

# Table bloat detection
cd tests/evaluation/test-table-bloat && docker-compose up

# Long-running query detection
cd tests/evaluation/test-long-running-query && docker-compose up
```

### Adding a New Detector

1. Add metrics collection in `collector/internal/adapter/postgres.go`
2. Create detector in `analyser/internal/detector/`
3. Register detector in `analyser/internal/orchestrator/orchestrator.go`
4. Add threshold to `analyser/internal/config/config.go`
5. Create action in `executor/internal/actions/` (if needed)
6. Wire action in `executor/internal/handler/detection_handler.go`
7. Write unit tests
8. Create test container in `tests/evaluation/`

## Documentation

- **Architecture Decision Records**: `docs/decisions/` - Rationale for technical decisions
- **Sprint Documentation**: `docs/sprints/` - Planning and retrospectives

## Limitations

- Single database monitoring per deployment (multi-database support planned)
- Some features require `pg_stat_statements` extension
- Query termination requires superuser or `pg_signal_backend` role

## Acknowledgements

Built as a Level 8 dissertation project at Atlantic Technological University, Galway.

## License

MIT License