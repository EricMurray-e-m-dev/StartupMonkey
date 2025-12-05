'use client';

import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CheckCircle, Clock, Loader2, XCircle, Wrench, Undo2, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useActions } from "@/hooks/useActions";
import { ActionResult } from "@/types/actions";
import { useState } from "react";

// Recommendation type definition
interface Recommendation {
    title: string;
    description: string;
    risk_level: 'safe' | 'medium' | 'advanced';
    steps?: string[];
    requires_restart?: boolean;
    requires_code_change?: boolean;
    deployable_action_type?: string;
}

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

    // Calculate stats (exclude rolled_back actions)
    const activeActions = actions.filter(a => a.status !== 'rolled_back');
    const queuedCount = activeActions.filter(a => a.status === 'queued').length;
    const executingCount = activeActions.filter(a => a.status === 'executing').length;
    const completedCount = activeActions.filter(a => a.status === 'completed').length;
    const failedCount = activeActions.filter(a => a.status === 'failed').length;

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
    const [isRollingBack, setIsRollingBack] = useState(false);

    const handleRollback = async () => {
        setIsRollingBack(true);
    
        try {
            const response = await fetch('/api/actions/rollback', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ action_id: action.action_id }),
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Rollback failed');
            }

            const result = await response.json();
            toast.success('Action rolled back successfully', {
                description: `${action.action_type} has been reverted`,
            });

            console.log("Rollback result: " + result);
        } catch (error) {
            console.error("Rollback error: " + error);
            toast.error('Rollback failed', {
                description: error instanceof Error ? error.message : 'Unknown Error',
            });
        } finally {
            setIsRollingBack(false);
        }
    }

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
        },
        rolled_back: { 
            variant: 'secondary' as const,
            icon: <Undo2 className="h-5 w-5 text-purple-600" />,
            bgClass: 'bg-purple-50 dark:bg-purple-950/20 border-purple-200 dark:border-purple-900'
        }
    };

    const config = statusConfig[action.status as keyof typeof statusConfig];

    // Check if this is a recommendation action
    const isRecommendation = action.action_type === 'recommendation' || action.action_type === 'cache_optimization_recommendation';

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
                    <div className="flex items-center gap-2">
                        <Badge variant={config.variant}>
                            {action.status}
                        </Badge>
                        {action.status === 'completed' && action.can_rollback && !isRecommendation && (
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={handleRollback}
                                disabled={isRollingBack}
                            >
                                {isRollingBack ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                    <Undo2 className="h-4 w-4" />
                                )}
                                <span className="ml-2">Rollback</span>
                            </Button>
                        )}
                    </div>
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

                {/* Recommendation-specific UI */}
                {isRecommendation && action.changes?.recommendations && Array.isArray(action.changes.recommendations) && (
                    <RecommendationDisplay 
                        recommendations={action.changes.recommendations as Recommendation[]}
                        databaseId={action.database_id}
                    />
                )}

                {/* Standard Changes (if not recommendation) */}
                {!isRecommendation && action.changes && Object.keys(action.changes).length > 0 && (
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

// Recommendation Display Component
function RecommendationDisplay({ 
    recommendations, 
    databaseId 
}: { 
    recommendations: Recommendation[];
    databaseId: string;
}) {
    const [deployingRedis, setDeployingRedis] = useState(false);

    const handleDeployRedis = async () => {
        setDeployingRedis(true);

        try {
            // Call Executor directly via HTTP (same pattern as rollback)
            const response = await fetch('http://localhost:8084/api/deploy-redis', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    database_id: databaseId,
                    port: '6380',
                    max_memory: '256mb',
                    eviction_policy: 'allkeys-lru'
                }),
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Deployment failed');
            }

            const result = await response.json();
            
            toast.success('Redis deployment started', {
                description: 'Redis container is being deployed. Check action queue for status.',
            });

            console.log('Deploy result:', result);
        } catch (error) {
            console.error('Deploy error:', error);
            toast.error('Deployment failed', {
                description: error instanceof Error ? error.message : 'Unknown error',
            });
        } finally {
            setDeployingRedis(false);
        }
    };

    return (
        <div className="space-y-4">
            {recommendations.map((rec, index) => (
                <RecommendationCard 
                    key={index}
                    recommendation={rec}
                    onDeployRedis={rec.risk_level === 'advanced' ? handleDeployRedis : undefined}
                    deployingRedis={deployingRedis}
                />
            ))}
        </div>
    );
}

// Individual Recommendation Card Component
function RecommendationCard({ 
    recommendation,
    onDeployRedis,
    deployingRedis 
}: { 
    recommendation: Recommendation;
    onDeployRedis?: () => void;
    deployingRedis?: boolean;
}) {
    const [showSteps, setShowSteps] = useState(false);

    const riskConfig = {
        safe: {
            borderColor: 'border-green-500',
            badgeVariant: 'default' as const,
            badgeClass: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100'
        },
        medium: {
            borderColor: 'border-yellow-500',
            badgeVariant: 'default' as const,
            badgeClass: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100'
        },
        advanced: {
            borderColor: 'border-red-500',
            badgeVariant: 'default' as const,
            badgeClass: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-100'
        }
    };

    const config = riskConfig[recommendation.risk_level] || riskConfig.medium;

    return (
        <div className={`border-l-4 ${config.borderColor} pl-4 py-2 space-y-3`}>
            {/* Header */}
            <div className="flex items-start justify-between">
                <div>
                    <div className="flex items-center gap-2 mb-1">
                        <h4 className="text-sm font-semibold">{recommendation.title}</h4>
                        <Badge className={config.badgeClass} variant={config.badgeVariant}>
                            {recommendation.risk_level}
                        </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">
                        {recommendation.description}
                    </p>
                </div>
            </div>

            {/* Warnings */}
            {recommendation.requires_restart && (
                <Alert className="bg-yellow-50 dark:bg-yellow-950/20 border-yellow-200">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                        Requires database restart (2-5 minutes downtime)
                    </AlertDescription>
                </Alert>
            )}

            {recommendation.requires_code_change && (
                <Alert className="bg-red-50 dark:bg-red-950/20 border-red-200">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                        Requires application code changes. Not recommended for beginners.
                    </AlertDescription>
                </Alert>
            )}

            {/* Steps */}
            {recommendation.steps && recommendation.steps.length > 0 && (
                <div>
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowSteps(!showSteps)}
                        className="text-xs"
                    >
                        {showSteps ? 'Hide' : 'View'} Instructions
                    </Button>

                    {showSteps && (
                        <ol className="mt-2 space-y-1 text-xs text-muted-foreground list-decimal list-inside">
                            {recommendation.steps.map((step: string, i: number) => (
                                <li key={i}>{step}</li>
                            ))}
                        </ol>
                    )}
                </div>
            )}

            {/* Deploy Redis Button (for advanced option) */}
            {onDeployRedis && recommendation.deployable_action_type === 'deploy_redis' && (
                <div className="flex gap-2">
                    <Button
                        onClick={onDeployRedis}
                        disabled={deployingRedis}
                        size="sm"
                        variant="destructive"
                    >
                        {deployingRedis ? (
                            <>
                                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                                Deploying...
                            </>
                        ) : (
                            <>
                                Deploy Redis (Advanced)
                            </>
                        )}
                    </Button>
                </div>
            )}
        </div>
    );
}