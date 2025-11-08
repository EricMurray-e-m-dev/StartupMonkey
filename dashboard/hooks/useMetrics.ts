import { useEffect, useRef, useState } from "react";

interface Metrics {
    DatabaseID: string;
    DatabaseType: string;
    Timestamp: number;
    HealthScore: number;
    ConnectionHealth: number;
    QueryHealth: number;
    StorageHealth: number;
    CacheHealth: number;
    AvailableMetrics: string[];
    Measurements: any;
    ExtendedMetrics: Record<string, number>;
    Labels: Record<string, string>;
}

export function useMetrics(interval: number = 5000) {
    const [metrics, setMetrics] = useState<Metrics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const isFirstLoad = useRef(true);

    useEffect(() => {
        const fetchMetrics = async () => {
            try {
                const response = await fetch('/api/metrics/latest')

                if (!response.ok) {
                    throw new Error('failed to fetch metrics')
                }

                const data = await response.json();
                setMetrics(data);

                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }

                if (error) setError(null);
            } catch (err) {
                const errMsg = err instanceof Error ? err.message : 'Unknown Error';

                if (isFirstLoad.current) {
                    setError(errMsg);
                    setLoading(false);
                }
            }
        };

        fetchMetrics();

        const intervalID = setInterval(fetchMetrics, interval);

        return () => clearInterval(intervalID)
    }, [interval]);

    return { metrics, loading, error};
}