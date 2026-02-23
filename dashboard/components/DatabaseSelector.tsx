'use client';

import { useDatabase } from '@/components/providers/DatabaseProvider';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Database, Circle } from 'lucide-react';

export function DatabaseSelector() {
    const { databases, selectedDatabaseId, setSelectedDatabaseId, loading } = useDatabase();

    if (loading) {
        return (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Database className="h-4 w-4" />
                <span>Loading...</span>
            </div>
        );
    }

    if (databases.length === 0) {
        return (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Database className="h-4 w-4" />
                <span>No databases configured</span>
            </div>
        );
    }

    return (
        <Select value={selectedDatabaseId || ''} onValueChange={setSelectedDatabaseId}>
            <SelectTrigger className="w-[200px]">
                <div className="flex items-center gap-2">
                    <Database className="h-4 w-4" />
                    <SelectValue placeholder="Select database" />
                </div>
            </SelectTrigger>
            <SelectContent>
                {databases.map((db) => (
                    <SelectItem key={db.database_id} value={db.database_id}>
                        <div className="flex items-center gap-2">
                            <HealthIndicator status={db.health_status} />
                            <span>{db.database_name}</span>
                            <span className="text-xs text-muted-foreground">
                                ({db.database_type})
                            </span>
                        </div>
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
}

function HealthIndicator({ status }: { status?: string }) {
    const color = status === 'healthy' 
        ? 'text-green-500' 
        : status === 'degraded' 
            ? 'text-yellow-500' 
            : 'text-muted-foreground';
    
    return <Circle className={`h-2 w-2 fill-current ${color}`} />;
}