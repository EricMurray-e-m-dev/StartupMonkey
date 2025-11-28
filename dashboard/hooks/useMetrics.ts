import { useState, useEffect } from 'react';
import { Metrics } from '@/types/metrics';

export function useMetrics(pollInterval: number = 5000) {
    const [metrics, setMetrics] = useState<Metrics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchMetrics = async () => {
            try {
                // Fetch from Dashboard's own API route (not directly from collector)
                const res = await fetch('/api/metrics/latest');
                
                if (!res.ok) {
                    throw new Error(`HTTP ${res.status}`);
                }

                const data = await res.json();
                
                // Only set metrics if data exists
                if (data && Object.keys(data).length > 0) {
                    setMetrics(data);
                    setError(null);
                }
                
                setLoading(false);
            } catch (err) {
                console.error('Failed to fetch metrics:', err);
                setError(err instanceof Error ? err.message : 'Unknown error');
                setLoading(false);
            }
        };

        // Fetch immediately
        fetchMetrics();

        // Then poll
        const interval = setInterval(fetchMetrics, pollInterval);

        return () => clearInterval(interval);
    }, [pollInterval]);

    return { metrics, loading, error };
}