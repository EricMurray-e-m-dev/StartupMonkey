'use client';

import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { CheckCircle, Clock, Loader2, XCircle, Wrench } from "lucide-react";
import { useActions } from "@/hooks/useActions";
import { ActionResult } from "@/types/actions";

export default function ActionsPage() {
    const { actions, loading } = useActions(5000);

    if (loading) {
        return (
            <div className="flex items-center justify-center h-96">
                <div className="animate-pulse text-muted-foreground">
                    Loading action queue...
                </div>
            </div>
        );
    }

    // Calculate stats
    const queuedCount = actions.filter(a => a.status === 'queued').length;
    const executingCount = actions.filter(a => a.status === 'executing').length;
    const completedCount = actions.filter(a => a.status === 'completed').length;
    const failedCount = actions.filter(a => a.status === 'failed').length;

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold">Action Queue</h1>
                <p className="text-muted-foreground">
                    Real-time autonomous action execution
                </p>
            </div>

            {/* Summary Cards */}
            <div className="grid gap-4 md:grid-cols-4">
                <SummaryCard
                    title="Queued"
                    value={queuedCount}
                    icon={<Clock className="h-4 w-4 text-blue-500" />}
                    variant="queued"
                />
                <SummaryCard
                    title="Executing"
                    value={executingCount}
                    icon={<Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />}
                    variant="executing"
                />
                <SummaryCard
                    title="Completed"
                    value={completedCount}
                    icon={<CheckCircle className="h-4 w-4 text-green-500" />}
                    variant="completed"
                />
                <SummaryCard
                    title="Failed"
                    value={failedCount}
                    icon={<XCircle className="h-4 w-4 text-red-500" />}
                    variant="failed"
                />
            </div>

            {/* No Actions State */}
            {actions.length === 0 && (
                <Alert>
                    <Wrench className="h-4 w-4" />
                    <AlertDescription>
                        No actions in queue. Actions will appear here when optimizations are executed.
                    </AlertDescription>
                </Alert>
            )}

            {/* Actions List */}
            <div className="space-y-4">
                {actions.map((action) => (
                    <ActionCard key={action.action_id} action={action} />
                ))}
            </div>
        </div>
    );
}

// Summary Card Component
function SummaryCard({ 
    title, 
    value, 
    icon
}: { 
    title: string; 
    value: number; 
    icon: React.ReactNode;
    variant: 'queued' | 'executing' | 'completed' | 'failed';
}) {
    return (
        <Card>
            <CardContent className="pt-6">
                <div className="flex items-center justify-between mb-2">
                    <div className="text-muted-foreground">{icon}</div>
                </div>
                <p className="text-xs text-muted-foreground mb-1">{title}</p>
                <p className="text-2xl font-bold">{value}</p>
            </CardContent>
        </Card>
    );
}

// Action Card Component
function ActionCard({ action }: { action: ActionResult }) {
    const statusConfig = {
        queued: {
            variant: 'secondary' as const,
            icon: <Clock className="h-5 w-5 text-blue-600" />,
            bgClass: 'bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-900'
        },
        executing: {
            variant: 'default' as const,
            icon: <Loader2 className="h-5 w-5 text-yellow-600 animate-spin" />,
            bgClass: 'bg-yellow-50 dark:bg-yellow-950/20 border-yellow-200 dark:border-yellow-900'
        },
        completed: {
            variant: 'default' as const,
            icon: <CheckCircle className="h-5 w-5 text-green-600" />,
            bgClass: 'bg-green-50 dark:bg-green-950/20 border-green-200 dark:border-green-900'
        },
        failed: {
            variant: 'destructive' as const,
            icon: <XCircle className="h-5 w-5 text-red-600" />,
            bgClass: 'bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-900'
        }
    };

    const config = statusConfig[action.status as keyof typeof statusConfig];

    return (
        <Card className={config.bgClass}>
            <CardHeader>
                <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                        {config.icon}
                        <div>
                            <CardTitle className="text-lg">
                                {action.action_type.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}
                            </CardTitle>
                            <CardDescription className="mt-1">
                                Action ID: {action.action_id}
                            </CardDescription>
                            <CardDescription className="mt-1">
                                {new Date(Number(action.created_at) * 1000).toLocaleString()}
                            </CardDescription>
                        </div>
                    </div>
                    <Badge variant={config.variant}>
                        {action.status}
                    </Badge>
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Message */}
                <div>
                    <p className="text-sm text-muted-foreground">
                        {action.message}
                    </p>
                </div>

                {/* Error (if failed) */}
                {action.error && (
                    <div className="border-l-2 border-red-500 pl-4">
                        <h4 className="text-sm font-semibold mb-1 text-red-600">Error</h4>
                        <p className="text-sm text-muted-foreground">
                            {action.error}
                        </p>
                    </div>
                )}

                {/* Changes (if completed) */}
                {action.changes && Object.keys(action.changes).length > 0 && (
                    <div className="border-l-2 border-green-500 pl-4">
                        <h4 className="text-sm font-semibold mb-2 text-green-600">Changes Applied</h4>
                        <div className="grid grid-cols-2 gap-2">
                            {Object.entries(action.changes).map(([key, value]) => (
                                <div key={key} className="text-xs">
                                    <span className="text-muted-foreground">{key}:</span>{' '}
                                    <span className="font-medium">{String(value)}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Database Info */}
                <div className="flex items-center gap-4 pt-2 border-t text-xs text-muted-foreground">
                    <span>Database: {action.database_id}</span>
                    <span>Detection: {action.detection_id}</span>
                </div>
            </CardContent>
        </Card>
    );
}