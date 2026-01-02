# StartupMonkey Quick Start

Autonomous database performance optimisation for PostgreSQL.

## Prerequisites

- Docker and Docker Compose installed
- A PostgreSQL database (v12+) accessible from where you run StartupMonkey
- Database user with permissions to:
  - Read system statistics (`pg_stat_statements`, `pg_stat_user_tables`)
  - Create indexes (`CREATE INDEX`)
  - Alter system config (`ALTER SYSTEM`)

## Installation

```bash
# Download release files
curl -O https://raw.githubusercontent.com/EricMurray-e-m-dev/StartupMonkey/main/docker-compose.release.yml
curl -O https://raw.githubusercontent.com/EricMurray-e-m-dev/StartupMonkey/main/.env.release.example

# Configure your database connection
cp .env.release.example .env
nano .env  # Edit with your database credentials

# Start StartupMonkey
docker-compose -f docker-compose.release.yml up -d

# View dashboard
open http://localhost:3000
```

## What Happens Next

1. **Collector** connects to your database and begins gathering metrics
2. **Analyser** monitors for performance issues (missing indexes, connection exhaustion, etc.)
3. **Executor** automatically applies safe optimisations when issues detected
4. **Dashboard** shows you what's happening in real-time

## Supported Optimisations

| Issue Detected | Automatic Action |
|----------------|------------------|
| Missing Index | Creates index using `CREATE INDEX CONCURRENTLY` (non-blocking) |
| Connection Pool Exhaustion | Deploys PgBouncer connection pooler |
| Cache Inefficiency | Tunes `effective_cache_size` parameter |
| High Latency Queries | Deploys Redis cache container |

All actions can be rolled back from the dashboard.

## Stopping StartupMonkey

```bash
docker-compose -f docker-compose.release.yml down
```

To also remove stored data:
```bash
docker-compose -f docker-compose.release.yml down -v
```

## Troubleshooting

**Collector won't start:**
- Check `DB_CONNECTION_STRING` is correct
- Ensure database is accessible from Docker network
- Verify user has required permissions

**No detections appearing:**
- Metrics are collected every 30s by default
- Check collector logs: `docker-compose -f docker-compose.release.yml logs collector`

**Dashboard not loading:**
- Wait 15-20 seconds for all services to start
- Check: `docker-compose -f docker-compose.release.yml ps`

## More Information

- GitHub: https://github.com/EricMurray-e-m-dev/StartupMonkey
- Issues: https://github.com/EricMurray-e-m-dev/StartupMonkey/issues
