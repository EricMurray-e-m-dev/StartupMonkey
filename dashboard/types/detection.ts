// Match Go enums exactly
export type DetectionCategory = 'query' | 'connection' | 'cache' | 'storage';
export type DetectionSeverity = 'info' | 'warning' | 'critical';

export interface Evidence {
    table_name?: string;
    column_name?: string;
    sequential_scans?: number;
    rows_read?: number;
    query_health?: number;
    detection_method?: string;

    active_connections?: number;
    max_connections?: number;
    connection_usage_percent?: number;
    
    cache_hit_rate?: number;
    cache_miss_rate?: number;

    database_size_mb?: number;
    disk_usage_percent?: number;
}

export interface ActionMetadata {
    table_name?: string;
    column_name?: string;
    priority?: string;
    autonomous?: boolean;
    manual_analysis?: boolean;
}

export interface Detection {
    id: string;
    detector_name: string;
    category: DetectionCategory;
    severity: DetectionSeverity;
    database_id: string;
    timestamp: number;
    title: string;
    description: string;
    evidence: Evidence;
    recommendation: string;
    action_type?: string;
    action_metadata?: ActionMetadata;
}