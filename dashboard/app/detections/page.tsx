import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { AlertTriangle, AlertCircle, Info } from "lucide-react";

export default function DetectionsPage() {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold">Detections</h1>
                <p className="text-muted-foreground">
                    Performance issues detected by analyser
                </p>
            </div>

            {/* Placeholer data */}
            <div className="space-y-4">
                <Alert variant={"destructive"}>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertTitle className="flex items-center gap-2">
                        Cache hit rate critically low
                        <Badge variant={"destructive"}>Critical</Badge>
                    </AlertTitle>
                    <AlertDescription>
                        Database cache rate is only 32%, meaning 68% of reads require disk I/O.
                        This significantly degrades query performance under load.
                        <div className="mt-2 text-xs">
                            <strong>Recommendation:</strong> increase shared_buffers to allocate more memory for caching.
                        </div>
                    </AlertDescription>
                </Alert>
    
                <Alert variant={"destructive"}>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertTitle className="flex items-center gap-2">
                        Sequential scans detected
                        <Badge variant={"destructive"}>Critical</Badge>
                    </AlertTitle>
                    <AlertDescription>
                        Queries are performing sequential scans on large tables without indexes.
                        <div className="mt-2 text-xs">
                            <strong>Recommendation:</strong> create indexes on frequently queried columns.
                        </div>
                    </AlertDescription>
                </Alert>

                <Alert>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertTitle className="flex items-center gap-2">
                        Connection pool utilisation high
                        <Badge variant={"secondary"}>Warning</Badge>
                    </AlertTitle>
                    <AlertDescription>
                        Using 85% of available connections. May need connection pooling
                        <div className="mt-2 text-xs">
                            <strong>Recommendation:</strong> deploy PgBouncer for connection pooling.
                        </div>
                    </AlertDescription>
                </Alert>
            </div>
        </div>
    );
}