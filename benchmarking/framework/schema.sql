-- Benchmarking Database Schema
-- Stores load test results from Locust + Collector metrics

-- Main benchmark runs table
CREATE TABLE IF NOT EXISTS benchmark_runs (
    run_id TEXT PRIMARY KEY,
    app_name TEXT NOT NULL,
    stage INTEGER NOT NULL,
    user_count INTEGER NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_seconds INTEGER NOT NULL,
    target_host TEXT NOT NULL,
    notes TEXT
);

-- Locust metrics per endpoint
CREATE TABLE IF NOT EXISTS locust_metrics (
    id SERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES benchmark_runs(run_id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    request_count INTEGER NOT NULL,
    failure_count INTEGER NOT NULL,
    median_response_time REAL NOT NULL,
    average_response_time REAL NOT NULL,
    min_response_time REAL NOT NULL,
    max_response_time REAL NOT NULL,
    p50_response_time REAL NOT NULL,
    p95_response_time REAL NOT NULL,
    p99_response_time REAL NOT NULL,
    requests_per_second REAL NOT NULL,
    failures_per_second REAL NOT NULL
);

-- Collector metrics (database health)
CREATE TABLE IF NOT EXISTS collector_metrics (
    id SERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES benchmark_runs(run_id) ON DELETE CASCADE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active_connections INTEGER,
    max_connections INTEGER,
    cache_hit_rate REAL,
    sequential_scans INTEGER,
    index_scans INTEGER,
    query_health REAL,
    connection_health REAL,
    cache_health REAL,
    overall_health REAL
);

-- Aggregated metrics per run (for easy querying)
CREATE TABLE IF NOT EXISTS run_summary (
    run_id TEXT PRIMARY KEY REFERENCES benchmark_runs(run_id) ON DELETE CASCADE,
    total_requests INTEGER NOT NULL,
    total_failures INTEGER NOT NULL,
    failure_rate REAL NOT NULL,
    avg_response_time REAL NOT NULL,
    p50_response_time REAL NOT NULL,
    p95_response_time REAL NOT NULL,
    p99_response_time REAL NOT NULL,
    requests_per_second REAL NOT NULL,
    db_cache_hit_rate REAL,
    db_overall_health REAL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_app_stage ON benchmark_runs(app_name, stage);
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_timestamp ON benchmark_runs(timestamp);
CREATE INDEX IF NOT EXISTS idx_locust_metrics_run_id ON locust_metrics(run_id);
CREATE INDEX IF NOT EXISTS idx_locust_metrics_endpoint ON locust_metrics(endpoint);
CREATE INDEX IF NOT EXISTS idx_collector_metrics_run_id ON collector_metrics(run_id);

-- View for easy comparison across stages
CREATE OR REPLACE VIEW stage_comparison AS
SELECT 
    br.app_name,
    br.stage,
    br.user_count,
    rs.avg_response_time,
    rs.p95_response_time,
    rs.p99_response_time,
    rs.failure_rate,
    rs.requests_per_second,
    rs.db_cache_hit_rate,
    rs.db_overall_health
FROM benchmark_runs br
JOIN run_summary rs ON br.run_id = rs.run_id
ORDER BY br.app_name, br.stage, br.user_count;

-- View for endpoint performance comparison
CREATE OR REPLACE VIEW endpoint_comparison AS
SELECT 
    br.app_name,
    br.stage,
    br.user_count,
    lm.endpoint,
    lm.p95_response_time,
    lm.requests_per_second,
    lm.failure_count,
    ROUND((lm.failure_count::NUMERIC / NULLIF(lm.request_count, 0) * 100), 2) as failure_rate_percent
FROM benchmark_runs br
JOIN locust_metrics lm ON br.run_id = lm.run_id
ORDER BY br.app_name, br.stage, br.user_count, lm.endpoint;