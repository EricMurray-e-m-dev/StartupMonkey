export interface Metrics {
    database_id: string;
    database_type: string;
    timestamp: number;
    health_score: number;
    connection_health: number;
    query_health: number;
    storage_health: number;
    cache_health: number;
    available_metrics: string[];
    measurements: Measurements;
    metric_deltas?: Record<string, number>;
    time_delta_seconds?: number;
    extended_metrics: Record<string, number>;
    labels: Record<string, string>;
}

export interface Measurements {
    active_connections?: number;
    idle_connections?: number;
    max_connections?: number;
    waiting_connections?: number;
    avg_query_latency_ms?: number;
    p50_query_latency_ms?: number;
    p95_query_latency_ms?: number;
    p99_query_latency_ms?: number;
    slow_query_count?: number;
    sequential_scans?: number;
    used_storage_bytes?: number;
    total_storage_bytes?: number;
    free_storage_bytes?: number;
    cache_hit_rate?: number;
    cache_hit_count?: number;
    cache_miss_count?: number;
}