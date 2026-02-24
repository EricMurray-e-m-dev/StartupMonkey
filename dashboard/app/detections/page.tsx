'use client';

import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle, AlertTriangle, Info, CheckCircle } from "lucide-react";
import { useDetections } from "@/hooks/useDetections";
import { useDatabase } from "@/components/providers/DatabaseProvider";
import { Detection } from "@/types/detection";

/** Format Unix timestamp (seconds) to locale string */
function formatTimestamp(timestamp: number): string {
    if (!timestamp || timestamp === 0) return 'Unknown';
    // Handle both seconds and milliseconds
    const ms = timestamp < 1e12 ? timestamp * 1000 : timestamp;
    return new Date(ms).toLocaleString();
}

export default function DetectionsPage() {
    const { detections, loading } = useDetections(5000);
    const { selectedDatabase, isAllSelected } = useDatabase();

    if (loading) {
        return (
            <div className="flex items-center justify-center h-96">
                <div className="animate-pulse text-muted-foreground">
                    Loading detections...
                </div>
            </div>
        );
    }

    const criticalCount = detections.filter(d => d.severity === 'critical').length;
    const warningCount = detections.filter(d => d.severity === 'warning').length;
    const infoCount = detections.filter(d => d.severity === 'info').length;

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold">Live Detections</h1>
                <p className="text-muted-foreground">
                    {isAllSelected 
                        ? 'Performance issues across all databases'
                        : selectedDatabase 
                            ? `Performance issues for ${selectedDatabase.database_name}`
                            : 'Real-time performance issue detection'
                    }
                </p>
            </div>

            {/* Summary Cards */}
            <div className="grid gap-4 md:grid-cols-4">
                <SummaryCard
                    title="Total Detections"
                    value={detections.length}
                    icon={<AlertCircle className="h-4 w-4" />}
                />
                <SummaryCard
                    title="Critical"
                    value={criticalCount}
                    icon={<AlertCircle className="h-4 w-4 text-red-500" />}
                    showAlert={criticalCount > 0}
                />
                <SummaryCard
                    title="Warning"
                    value={warningCount}
                    icon={<AlertTriangle className="h-4 w-4 text-yellow-500" />}
                />
                <SummaryCard
                    title="Info"
                    value={infoCount}
                    icon={<Info className="h-4 w-4 text-blue-500" />}
                />
            </div>

            {/* No Detections State */}
            {detections.length === 0 && (
                <Alert>
                    <CheckCircle className="h-4 w-4" />
                    <AlertDescription>
                        {isAllSelected
                            ? 'No issues detected across any database. All databases are running smoothly!'
                            : selectedDatabase
                                ? `No issues detected for ${selectedDatabase.database_name}. Your database is running smoothly!`
                                : 'No issues detected. Your database is running smoothly!'
                        }
                    </AlertDescription>
                </Alert>
            )}

            {/* Detections List */}
            <div className="space-y-4">
                {detections.map((detection) => (
                    <DetectionCard key={detection.id} detection={detection} showDatabaseBadge={isAllSelected} />
                ))}
            </div>
        </div>
    );
}

function SummaryCard({ 
    title, 
    value, 
    icon, 
    showAlert = false
}: { 
    title: string; 
    value: number; 
    icon: React.ReactNode;
    showAlert?: boolean;
}) {
    return (
        <Card>
            <CardContent className="pt-6">
                <div className="flex items-center justify-between mb-2">
                    <div className="text-muted-foreground">{icon}</div>
                    {showAlert && (
                        <Badge variant="destructive" className="text-xs">!</Badge>
                    )}
                </div>
                <p className="text-xs text-muted-foreground mb-1">{title}</p>
                <p className="text-2xl font-bold">{value}</p>
            </CardContent>
        </Card>
    );
}

function DetectionCard({ detection, showDatabaseBadge = false }: { detection: Detection; showDatabaseBadge?: boolean }) {
    const severityConfig = {
        critical: {
            variant: 'destructive' as const,
            icon: <AlertCircle className="h-5 w-5" />,
            bgClass: 'bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-900'
        },
        warning: {
            variant: 'default' as const,
            icon: <AlertTriangle className="h-5 w-5 text-yellow-600" />,
            bgClass: 'bg-yellow-50 dark:bg-yellow-950/20 border-yellow-200 dark:border-yellow-900'
        },
        info: {
            variant: 'secondary' as const,
            icon: <Info className="h-5 w-5 text-blue-600" />,
            bgClass: 'bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-900'
        }
    };

    const config = severityConfig[detection.severity as keyof typeof severityConfig] || severityConfig.info;

    return (
        <Card className={config.bgClass}>
            <CardHeader>
                <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                        {config.icon}
                        <div>
                            <div className="flex items-center gap-2">
                                <CardTitle className="text-lg">{detection.title}</CardTitle>
                                {showDatabaseBadge && (
                                    <Badge variant="outline" className="text-xs">
                                        {detection.database_id}
                                    </Badge>
                                )}
                            </div>
                            <CardDescription className="mt-1">
                                {formatTimestamp(detection.timestamp)}
                            </CardDescription>
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <Badge variant={config.variant}>
                            {detection.severity}
                        </Badge>
                        <Badge variant="outline">
                            {detection.category}
                        </Badge>
                    </div>
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Description */}
                <div>
                    <h4 className="text-sm font-semibold mb-1">Issue</h4>
                    <p className="text-sm text-muted-foreground">
                        {detection.description}
                    </p>
                </div>

                {/* Recommendation */}
                {detection.recommendation && (
                    <div>
                        <h4 className="text-sm font-semibold mb-1">Recommendation</h4>
                        <p className="text-sm text-muted-foreground">
                            {detection.recommendation}
                        </p>
                    </div>
                )}

                {/* Evidence */}
                {detection.evidence && Object.keys(detection.evidence).length > 0 && (
                    <div>
                        <h4 className="text-sm font-semibold mb-2">Evidence</h4>
                        <div className="grid grid-cols-2 gap-2">
                            {Object.entries(detection.evidence).map(([key, value]) => (
                                <div key={key} className="text-xs">
                                    <span className="text-muted-foreground">{key}:</span>{' '}
                                    <span className="font-medium">{String(value)}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Footer: Action Type + Database Info (only when not viewing all) */}
                {!showDatabaseBadge && (
                    <div className="flex items-center justify-between pt-2 border-t text-xs text-muted-foreground">
                        <span>Database: {detection.database_id}</span>
                        {detection.action_type && (
                            <Badge variant="outline" className="text-xs">
                                Action: {detection.action_type}
                            </Badge>
                        )}
                    </div>
                )}

                {/* Action type only when viewing all */}
                {showDatabaseBadge && detection.action_type && (
                    <div className="flex items-center justify-end pt-2 border-t text-xs text-muted-foreground">
                        <Badge variant="outline" className="text-xs">
                            Action: {detection.action_type}
                        </Badge>
                    </div>
                )}
            </CardContent>
        </Card>
    );
}