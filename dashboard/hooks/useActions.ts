import { useEffect, useState, useRef } from "react";
import { ActionResult } from "@/types/actions";

export function useActions(interval: number = 5000) {
    const [actions, setActions] = useState<ActionResult[]>([]);
    const [loading, setLoading] = useState(true);
    const isFirstLoad = useRef(true);

    useEffect(() => {
        const fetchActions = async () => {
            try {
                const response = await fetch("/api/actions/latest")

                if (!response.ok) {
                    throw new Error("Failed to fetch actions")
                }

                const data = await response.json();


                if (Array.isArray(data)) {
                    setActions(data);
                    
                    if (isFirstLoad.current) {
                        setLoading(false);
                        isFirstLoad.current = false;
                    }
                } else if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }

            } catch (error) {
                console.warn("Failed to fetch actions: ", error)
                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            }
        };

        fetchActions();
        const intervalID = setInterval(fetchActions, interval);

        return () => clearInterval(intervalID);
    }, [interval]);

    return { actions, loading };
}