'use client';

import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle, AlertTriangle, Info, CheckCircle } from "lucide-react";
import { useDetections } from "@/hooks/useDetections";
import { Detection } from "@/types/detection";

export default function DetectionsPage() {
    const { detections, loading } = useDetections(5000);

    if (loading) {
        return (
            <div className="flex items-center justify-center h-96">
                <div className="animate-pulse text-muted-foreground">
                    Loading detections...
                </div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold">Live Detections</h1>
                <p className="text-muted-foreground">
                    Real-time performance issue detection
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
                    value={detections.filter(d => d.severity === 'critical').length}
                    icon={<AlertCircle className="h-4 w-4 text-red-500" />}
                    variant="critical"
                />
                <SummaryCard
                    title="Warning"
                    value={detections.filter(d => d.severity === 'warning').length}
                    icon={<AlertTriangle className="h-4 w-4 text-yellow-500" />}
                    variant="warning"
                />
                <SummaryCard
                    title="Info"
                    value={detections.filter(d => d.severity === 'info').length}
                    icon={<Info className="h-4 w-4 text-blue-500" />}
                    variant="info"
                />
            </div>

            {/* No Detections State */}
            {detections.length === 0 && (
                <Alert>
                    <CheckCircle className="h-4 w-4" />
                    <AlertDescription>
                        No issues detected. Your database is running smoothly!
                    </AlertDescription>
                </Alert>
            )}

            {/* Detections List */}
            <div className="space-y-4">
                {detections.map((detection) => (
                    <DetectionCard key={detection.id} detection={detection} />
                ))}
            </div>
        </div>
    );
}

// Summary Card Component
function SummaryCard({ 
    title, 
    value, 
    icon, 
    variant 
}: { 
    title: string; 
    value: number; 
    icon: React.ReactNode;
    variant?: 'critical' | 'warning' | 'info';
}) {
    return (
        <Card>
            <CardContent className="pt-6">
                <div className="flex items-center justify-between mb-2">
                    <div className="text-muted-foreground">{icon}</div>
                    {variant === 'critical' && value > 0 && (
                        <Badge variant="destructive" className="text-xs">!</Badge>
                    )}
                </div>
                <p className="text-xs text-muted-foreground mb-1">{title}</p>
                <p className="text-2xl font-bold">{value}</p>
            </CardContent>
        </Card>
    );
}

// Detection Card Component
function DetectionCard({ detection }: { detection: Detection }) {
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
                            <CardTitle className="text-lg">{detection.title}</CardTitle>
                            <CardDescription className="mt-1">
                                {new Date(detection.timestamp * 1000).toLocaleString()}
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
                <div>
                    <h4 className="text-sm font-semibold mb-1">Recommendation</h4>
                    <p className="text-sm text-muted-foreground">
                        {detection.recommendation}
                    </p>
                </div>

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

                {/* Action Type */}
                {detection.action_type && (
                    <div className="flex items-center gap-2 pt-2 border-t">
                        <Badge variant="outline" className="text-xs">
                            Recommended Action: {detection.action_type}
                        </Badge>
                    </div>
                )}
            </CardContent>
        </Card>
    );
}