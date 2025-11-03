import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

const mockActions = [
    {
        id: "action-001",
        type: "create_index",
        status: "completed",
        database: "postgres-1",
        timestamp: "2 minutes ago",
    },
    {
        id: "action-002",
        type: "increase_shared_buffers",
        status: "queued",
        database: "postgres-1",
        timestamp: "Just now",
    },
    {
        id: "action-003",
        type: "deploy_pgbouncer",
        status: "pending_approval",
        database: "postgres-1",
        timestamp: "30 seconds ago",
    },
];

export default function ActionsPage() {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold">Actions</h1>
                <p className="text-muted-foreground">
                    Optimisation actions queued and executed
                </p>
            </div>

            {/* Placeholder data */}
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Action ID</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead>Database</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Time</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {mockActions.map((action) => (
                        <TableRow key={action.id}>
                            <TableCell className="font-mono text-xs">
                                {action.id}
                            </TableCell>
                            <TableCell>{action.type}</TableCell>
                            <TableCell>{action.database}</TableCell>
                            <TableCell> 
                                {action.status === "completed" && (
                                    <Badge variant={"default"}>Completed</Badge>
                                )}
                                {action.status === "queued" && (
                                    <Badge variant={"secondary"}>Queued</Badge>
                                )}
                                {action.status === "pending_approval" && (
                                    <Badge variant={"outline"}>Pending Approval</Badge>
                                )}
                            </TableCell>
                            <TableCell className="text-muted-foreground">
                                {action.timestamp}
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </div>
    );
}