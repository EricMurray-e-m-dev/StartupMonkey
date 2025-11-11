import { useEffect, useState, useRef } from "react";
import { Detection } from "@/types/detection";

export function useDetections(interval: number = 5000) {
    const [detections, setDetections] = useState<Detection[]>([]);
    const [loading, setLoading] = useState(true);
    const isFirstLoad = useRef(true);

    useEffect(() => {
        const fetchDetections = async () => {
            try {
                const response = await fetch("/api/detections/latest");

                if (!response.ok) {
                    throw new Error("failed to fetch detections");
                }

                const data = await response.json();

                if (Array.isArray(data) && data.length > 0) {
                    setDetections(data);

                    if (isFirstLoad.current) {
                        setLoading(false);
                        isFirstLoad.current = false;
                    }
                } else if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            } catch (error) {
                console.warn("Failed to fetch detections:", error)
                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            }
        };

        fetchDetections();
        const intervalID = setInterval(fetchDetections, interval);

        return () => clearInterval(intervalID);
    }, [interval]);

    return { detections, loading }
}