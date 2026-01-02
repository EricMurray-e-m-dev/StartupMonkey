# Decision 014: Centralised Configuration Strategy

**Date:** 2025-12-01  
**Status:** Accepted

## Context

As StartupMonkey grew to 5 services, configuration management became unwieldy:

**Problems:**
1. Each service had its own environment variable handling
2. Inconsistent naming: `DB_HOST` vs `POSTGRES_HOST` vs `DATABASE_HOST`
3. Repeated boilerplate for parsing, validation, defaults
4. Docker Compose environment sections growing large
5. Easy to miss a variable when adding new config

**Configuration needed:**
- Database connection details (host, port, user, password, dbname)
- Service ports and addresses
- NATS connection string
- Redis connection string
- Feature flags and thresholds

## Decision

Implement a **standardised configuration pattern** across all Go services:

**Structure:**
```
service/
├── internal/
│   └── config/
│       └── config.go    # Single config file per service
├── internal/
│   └── orchestrator/
│       └── orchestrator.go  # Uses config
└── cmd/
    └── service/
        └── main.go      # Loads config, passes to orchestrator
```

**Config Pattern:**
```go
// config/config.go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    NATS     NATSConfig
    // ... service-specific config
}

func Load() (*Config, error) {
    return &Config{
        Server: ServerConfig{
            Port: getEnv("SERVER_PORT", "8080"),
        },
        Database: DatabaseConfig{
            Host: getEnv("DB_HOST", "localhost"),
            // ...
        },
    }, nil
}
```

**Standardised Environment Variables:**
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database
- `NATS_URL` - NATS connection
- `REDIS_URL` - Redis connection
- `KNOWLEDGE_GRPC_ADDR` - Knowledge service address
- `SERVER_PORT` - HTTP/health check port
- `GRPC_PORT` - gRPC server port

**Single .env File:**
Root-level `.env` file with all configuration, loaded by Docker Compose.

## Consequences

**Positive:**
- Consistent config loading across all services
- Single source of truth (root `.env` file)
- Standardised variable names
- Easy to see all configuration in one place
- Validation happens at startup (fail fast)
- Orchestrator receives config, doesn't know about environment

**Negative:**
- Refactoring effort to standardise existing services
- All services must follow the pattern (enforcement via code review)
- Large `.env` file for full system

**Trade-offs Accepted:**
- Upfront refactoring cost is worth long-term maintainability
- Pattern must be documented for consistency

## Alternatives Considered

**Config Files (YAML/JSON):**
- Rejected: Environment variables are standard for containerised apps, better Docker/K8s integration

**Centralised Config Service:**
- Rejected: Overkill for our scale, adds startup dependency

**Viper/Similar Library:**
- Considered: Decided simple `os.Getenv` wrapper sufficient, avoids dependency

**Per-Service .env Files:**
- Rejected: Duplication, easy to get out of sync
