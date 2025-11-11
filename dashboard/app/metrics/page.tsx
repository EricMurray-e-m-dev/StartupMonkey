'use client';

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle, Database, Activity, TrendingUp } from "lucide-react";
import { useMetrics } from '@/hooks/useMetrics';

export default function MetricsPage() {
    const { metrics, loading, error } = useMetrics(5000); // Poll every 5s

    if (loading) {
        return (
            <div className="flex items-center justify-center h-96">
                <div className="animate-pulse text-muted-foreground">
                    Loading live feed...
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                    Failed to load live feed: {error}
                </AlertDescription>
            </Alert>
        );
    }

    if (!metrics) {
        return (
            <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                    No metrics available yet. Make sure Collector is running.
                </AlertDescription>
            </Alert>
        );
    }

    // Calculate percentages
    const connectionPercent = Math.round(metrics.ConnectionHealth * 100);
    const queryPercent = Math.round(metrics.QueryHealth * 100);
    const cachePercent = Math.round(metrics.CacheHealth * 100);
    const storagePercent = Math.round(metrics.StorageHealth * 100);
    const overallPercent = Math.round(metrics.HealthScore * 100);

    // Get measurements
    const measurements = metrics.Measurements;
    const activeConns = measurements.ActiveConnections || 0;
    const maxConns = measurements.MaxConnections || 100;
    const seqScans = measurements.SequentialScans || 0;
    const cacheHitRate = measurements.CacheHitRate || 0;

    // Database size from extended metrics
    const dbSizeMB = metrics.ExtendedMetrics?.['pg.database_size_mb']?.toFixed(2) || '0';

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold">Real-time Metrics</h1>
                <p className="text-muted-foreground">
                    Database: {metrics.DatabaseID} ({metrics.DatabaseType})
                </p>
                <p className="text-xs text-muted-foreground">
                    Last updated: {new Date(metrics.Timestamp * 1000).toLocaleTimeString()}
                </p>
            </div>

            {/* Overall Health Score - Big Card */}
            <Card className="border-2">
                <CardContent className="pt-6">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">Overall Health Score</p>
                            <p className="text-4xl font-bold mt-2">{overallPercent}%</p>
                        </div>
                        <Activity className="h-12 w-12 text-muted-foreground" />
                    </div>
                    <Progress value={overallPercent} className="h-3 mt-4" />
                </CardContent>
            </Card>

            {/* Health Scores Grid */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <HealthCard
                    title="Connection Health"
                    value={connectionPercent}
                    detail={`${activeConns}/${maxConns} active`}
                />
                <HealthCard
                    title="Query Health"
                    value={queryPercent}
                    detail={seqScans > 0 ? `${seqScans} seq scans` : 'No issues'}
                    warning={seqScans > 10}
                />
                <HealthCard
                    title="Cache Health"
                    value={cachePercent}
                    detail={`${(cacheHitRate * 100).toFixed(1)}% hit rate`}
                    warning={cachePercent < 50}
                />
                <HealthCard
                    title="Storage Health"
                    value={storagePercent}
                    detail={`${dbSizeMB} MB used`}
                />
            </div>

            {/* Key Metrics Cards */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <MetricCard
                    icon={<Database className="h-4 w-4" />}
                    title="Active Connections"
                    value={activeConns.toString()}
                    max={maxConns.toString()}
                />
                <MetricCard
                    icon={<AlertCircle className="h-4 w-4" />}
                    title="Sequential Scans"
                    value={seqScans.toString()}
                    warning={seqScans > 10}
                />
                <MetricCard
                    icon={<TrendingUp className="h-4 w-4" />}
                    title="Cache Hit Rate"
                    value={`${(cacheHitRate * 100).toFixed(1)}%`}
                />
                <MetricCard
                    icon={<Database className="h-4 w-4" />}
                    title="Database Size"
                    value={`${dbSizeMB} MB`}
                />
            </div>

            {/* Table Detection Info */}
            {metrics.Labels?.['pg.worst_seq_scan_table'] && (
                <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                        High sequential scans detected on table <strong>{metrics.Labels['pg.worst_seq_scan_table']}</strong>.
                        {metrics.Labels['pg.recommended_index_column'] && (
                            <> Recommended index: <strong>{metrics.Labels['pg.recommended_index_column']}</strong></>
                        )}
                    </AlertDescription>
                </Alert>
            )}

            {/* Raw Data (for debugging) */}
            <Card>
                <CardHeader>
                    <CardTitle>Raw Metrics</CardTitle>
                </CardHeader>
                <CardContent>
                    <pre className="text-xs overflow-auto max-h-96 p-4 bg-muted rounded">
                        {JSON.stringify(metrics, null, 2)}
                    </pre>
                </CardContent>
            </Card>
        </div>
    );
}

// Helper Components
function HealthCard({ 
    title, 
    value, 
    detail, 
    warning 
}: { 
    title: string; 
    value: number; 
    detail: string;
    warning?: boolean;
}) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center justify-between">
                    <span className="text-sm font-medium">{title}</span>
                    {warning && (
                        <Badge variant="destructive" className="text-xs">
                            Warning
                        </Badge>
                    )}
                </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
                <div className="flex items-center justify-between">
                    <span className="text-3xl font-bold">{value}%</span>
                </div>
                <Progress 
                    value={value} 
                    className={`h-2 ${warning ? 'bg-red-200' : ''}`}
                />
                <p className="text-xs text-muted-foreground">
                    {detail}
                </p>
            </CardContent>
        </Card>
    );
}

function MetricCard({
    icon,
    title,
    value,
    max,
    warning
}: {
    icon: React.ReactNode;
    title: string;
    value: string;
    max?: string;
    warning?: boolean;
}) {
    return (
        <Card>
            <CardContent className="pt-6">
                <div className="flex items-center justify-between mb-2">
                    <div className="text-muted-foreground">{icon}</div>
                    {warning && (
                        <Badge variant="destructive" className="text-xs">!</Badge>
                    )}
                </div>
                <p className="text-xs text-muted-foreground mb-1">{title}</p>
                <p className="text-2xl font-bold">
                    {value}
                    {max && <span className="text-sm text-muted-foreground">/{max}</span>}
                </p>
            </CardContent>
        </Card>
    );
}