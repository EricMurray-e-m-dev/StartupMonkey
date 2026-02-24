import { useState, useEffect } from 'react';
import { useDatabase, ALL_DATABASES } from '@/components/providers/DatabaseProvider';
import { Metrics } from '@/types/metrics';

export function useMetrics(pollInterval: number = 5000) {
    const { selectedDatabaseId } = useDatabase();
    const [metrics, setMetrics] = useState<Metrics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const isAllSelected = selectedDatabaseId === ALL_DATABASES;

    useEffect(() => {
        // Metrics can't be aggregated across databases
        if (isAllSelected) {
            setMetrics(null);
            setLoading(false);
            setError(null);
            return;
        }

        const fetchMetrics = async () => {
            try {
                const params = selectedDatabaseId 
                    ? `?database_id=${selectedDatabaseId}` 
                    : '';
                const res = await fetch(`/api/metrics/latest${params}`);
                
                if (!res.ok) {
                    throw new Error(`HTTP ${res.status}`);
                }

                const data = await res.json();
                
                if (data && Object.keys(data).length > 0) {
                    setMetrics(data);
                    setError(null);
                } else {
                    setMetrics(null);
                }
                
                setLoading(false);
            } catch (err) {
                console.error('Failed to fetch metrics:', err);
                setError(err instanceof Error ? err.message : 'Unknown error');
                setLoading(false);
            }
        };

        setLoading(true);
        fetchMetrics();
        const interval = setInterval(fetchMetrics, pollInterval);

        return () => clearInterval(interval);
    }, [pollInterval, selectedDatabaseId, isAllSelected]);

    return { metrics, loading, error, isAllSelected };
}