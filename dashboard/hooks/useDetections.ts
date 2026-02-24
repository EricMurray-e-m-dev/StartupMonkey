import { useEffect, useState, useRef } from "react";
import { useDatabase, ALL_DATABASES } from "@/components/providers/DatabaseProvider";
import { Detection } from "@/types/detection";

export function useDetections(interval: number = 5000) {
    const { selectedDatabaseId } = useDatabase();
    const [detections, setDetections] = useState<Detection[]>([]);
    const [loading, setLoading] = useState(true);
    const isFirstLoad = useRef(true);

    useEffect(() => {
        const fetchDetections = async () => {
            try {
                const params = selectedDatabaseId && selectedDatabaseId !== ALL_DATABASES
                    ? `?database_id=${selectedDatabaseId}` 
                    : '';
                const response = await fetch(`/api/detections/latest${params}`);

                if (!response.ok) {
                    throw new Error("Failed to fetch detections");
                }

                const data = await response.json();

                if (Array.isArray(data)) {
                    setDetections(data);
                }

                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            } catch (error) {
                console.warn("Failed to fetch detections:", error);
                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            }
        };

        // Reset on database change
        setDetections([]);
        isFirstLoad.current = true;
        setLoading(true);

        fetchDetections();
        const intervalID = setInterval(fetchDetections, interval);

        return () => clearInterval(intervalID);
    }, [interval, selectedDatabaseId]);

    return { detections, loading };
}