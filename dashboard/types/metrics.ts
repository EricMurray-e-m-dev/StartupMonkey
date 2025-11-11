export interface Metrics {
    DatabaseID: string;
    DatabaseType: string;
    Timestamp: number;
    HealthScore: number;
    ConnectionHealth: number;
    QueryHealth: number;
    StorageHealth: number;
    CacheHealth: number;
    AvailableMetrics: string[];
    Measurements: Record<string, number | null>;
    ExtendedMetrics: Record<string, number>;
    Labels: Record<string, string>;
}