'use client';

import { useState } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { AlertCircle, Database, Activity, TrendingUp, ChevronDown, ChevronUp, Layers } from "lucide-react";
import { useMetrics } from '@/hooks/useMetrics';
import { useDatabase } from '@/components/providers/DatabaseProvider';

/** Format Unix timestamp (seconds) to locale string */
function formatTimestamp(timestamp: number): string {
    if (!timestamp || timestamp === 0) return 'Unknown';
    const ms = timestamp < 1e12 ? timestamp * 1000 : timestamp;
    return new Date(ms).toLocaleString();
}

export default function MetricsPage() {
    const { metrics, loading, error, isAllSelected } = useMetrics(5000);
    const { selectedDatabase, databases } = useDatabase();
    const [showRawMetrics, setShowRawMetrics] = useState(false);

    // Show prompt to select a database when "All" is selected
    if (isAllSelected) {
        return (
            <div className="space-y-6">
                <div>
                    <h1 className="text-3xl font-bold">Real-time Metrics</h1>
                    <p className="text-muted-foreground">
                        Select a database to view metrics
                    </p>
                </div>

                <Alert>
                    <Layers className="h-4 w-4" />
                    <AlertDescription>
                        Metrics cannot be aggregated across databases. Please select a specific database from the dropdown to view its metrics.
                    </AlertDescription>
                </Alert>

                {/* Show database list as quick links */}
                {databases.length > 0 && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-lg">Available Databases</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="grid gap-2 md:grid-cols-2 lg:grid-cols-3">
                                {databases.map((db) => (
                                    <div 
                                        key={db.database_id}
                                        className="flex items-center gap-2 p-3 rounded-lg border bg-muted/50"
                                    >
                                        <Database className="h-4 w-4 text-muted-foreground" />
                                        <div>
                                            <p className="text-sm font-medium">{db.database_name}</p>
                                            <p className="text-xs text-muted-foreground">{db.database_type}</p>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </CardContent>
                    </Card>
                )}
            </div>
        );
    }

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
            <div className="space-y-6">
                <div>
                    <h1 className="text-3xl font-bold">Real-time Metrics</h1>
                    <p className="text-muted-foreground">
                        {selectedDatabase 
                            ? `${selectedDatabase.database_name} (${selectedDatabase.database_type})`
                            : 'Select a database'
                        }
                    </p>
                </div>
                <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                        {selectedDatabase
                            ? `No metrics available for ${selectedDatabase.database_name}. Make sure Collector is running.`
                            : 'No metrics available yet. Make sure Collector is running.'
                        }
                    </AlertDescription>
                </Alert>
            </div>
        );
    }

    // Calculate percentages
    const connectionPercent = Math.round(metrics.connection_health * 100);
    const queryPercent = Math.round(metrics.query_health * 100);
    const cachePercent = Math.round(metrics.cache_health * 100);
    const storagePercent = Math.round(metrics.storage_health * 100);
    const overallPercent = Math.round(metrics.health_score * 100);

    // Get measurements
    const measurements = metrics.measurements || {};
    const activeConns = measurements.active_connections || 0;
    const maxConns = measurements.max_connections || 100;
    const seqScans = measurements.sequential_scans || 0;
    const cacheHitRate = measurements.cache_hit_rate || 0;

    // Database size from extended metrics
    const extendedMetrics = metrics.extended_metrics || {};
    const dbSizeMB = extendedMetrics['pg.database_size_mb']?.toFixed(2) || '0';

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold">Real-time Metrics</h1>
                <p className="text-muted-foreground">
                    {selectedDatabase 
                        ? `${selectedDatabase.database_name} (${selectedDatabase.database_type})`
                        : `${metrics.database_id} (${metrics.database_type})`
                    }
                </p>
                <p className="text-xs text-muted-foreground">
                    Last updated: {formatTimestamp(metrics.timestamp)}
                </p>
            </div>

            {/* Overall Health Score */}
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
            {metrics.labels?.['pg.worst_seq_scan_table'] && (
                <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                        High sequential scans detected on table <strong>{metrics.labels['pg.worst_seq_scan_table']}</strong>.
                        {metrics.labels['pg.recommended_index_column'] && (
                            <> Recommended index: <strong>{metrics.labels['pg.recommended_index_column']}</strong></>
                        )}
                    </AlertDescription>
                </Alert>
            )}

            {/* Raw Data (collapsible, for debugging) */}
            <Card>
                <CardHeader className="cursor-pointer" onClick={() => setShowRawMetrics(!showRawMetrics)}>
                    <div className="flex items-center justify-between">
                        <CardTitle className="text-sm">Raw Metrics</CardTitle>
                        <Button variant="ghost" size="sm">
                            {showRawMetrics ? (
                                <ChevronUp className="h-4 w-4" />
                            ) : (
                                <ChevronDown className="h-4 w-4" />
                            )}
                        </Button>
                    </div>
                </CardHeader>
                {showRawMetrics && (
                    <CardContent>
                        <pre className="text-xs overflow-auto max-h-96 p-4 bg-muted rounded">
                            {JSON.stringify(metrics, null, 2)}
                        </pre>
                    </CardContent>
                )}
            </Card>
        </div>
    );
}

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