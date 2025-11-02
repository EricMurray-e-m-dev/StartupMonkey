import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";

export default function MetricsPage() {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold">Metrics</h1>
                <p className="text-muted-foreground">
                    Real-time database health metrics
                </p>
            </div>

            {/* Placeholder data */}
            <div className="grid gap-4 md:grid-cols-2">
                <Card>
                    <CardHeader>
                        <CardTitle>Connection Health</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Score</span>
                            <span className="text-2xl font-bold">92%</span>
                        </div>
                        <Progress value={92} className="h-2" />
                        <p className="text-xs text-muted foreground">
                            5/100 connections active
                        </p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>Query Performance</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Score</span>
                            <span className="text-2xl font-bold">78%</span>
                        </div>
                        <Progress value={78} className="h-2" />
                        <p className="text-xs text-muted foreground">
                            Avg latency: 45ms
                        </p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>Cache Efficiency</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Score</span>
                            <span className="text-2xl font-bold">32%</span>
                        </div>
                        <Progress value={32} className="h-2" />
                        <p className="text-xs text-muted foreground">
                            Hit rate: 32% (low!)
                        </p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>Storage Health</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Score</span>
                            <span className="text-2xl font-bold">95%</span>
                        </div>
                        <Progress value={95} className="h-2" />
                        <p className="text-xs text-muted foreground">
                            65% disk used
                        </p>
                    </CardContent>
                </Card>                                 
            </div>
        </div>
    )
}