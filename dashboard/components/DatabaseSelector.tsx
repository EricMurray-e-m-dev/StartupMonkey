'use client';

import { useDatabase, ALL_DATABASES } from '@/components/providers/DatabaseProvider';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Database, Circle, Layers } from 'lucide-react';

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
        <Select value={selectedDatabaseId || ALL_DATABASES} onValueChange={setSelectedDatabaseId}>
            <SelectTrigger className="w-[200px]">
                <div className="flex items-center gap-2 overflow-hidden">
                    <Database className="h-4 w-4 shrink-0" />
                    <span className="truncate">
                        <SelectValue placeholder="Select database" />
                    </span>
                </div>
            </SelectTrigger>
            <SelectContent>
                {/* All Databases Option */}
                <SelectItem value={ALL_DATABASES}>
                    <div className="flex items-center gap-2">
                        <Layers className="h-3 w-3" />
                        <span>All Databases</span>
                    </div>
                </SelectItem>

                {/* Individual Databases */}
                {databases.map((db) => (
                    <SelectItem key={db.database_id} value={db.database_id}>
                        <div className="flex items-center gap-2">
                            <HealthIndicator status={db.health_status} />
                            <span className="truncate max-w-[120px]">{db.database_name}</span>
                            <span className="text-xs text-muted-foreground shrink-0">
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