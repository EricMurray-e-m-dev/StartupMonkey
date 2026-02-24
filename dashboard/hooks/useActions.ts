import { useEffect, useState, useRef } from "react";
import { useDatabase, ALL_DATABASES } from "@/components/providers/DatabaseProvider";
import { ActionResult } from "@/types/actions";

export function useActions(interval: number = 5000) {
    const { selectedDatabaseId } = useDatabase();
    const [actions, setActions] = useState<ActionResult[]>([]);
    const [loading, setLoading] = useState(true);
    const isFirstLoad = useRef(true);

    useEffect(() => {
        const fetchActions = async () => {
            try {
                const params = selectedDatabaseId && selectedDatabaseId !== ALL_DATABASES
                    ? `?database_id=${selectedDatabaseId}` 
                    : '';
                const response = await fetch(`/api/actions/latest${params}`);

                if (!response.ok) {
                    throw new Error("Failed to fetch actions");
                }

                const data = await response.json();

                if (Array.isArray(data)) {
                    setActions(data);
                }

                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            } catch (error) {
                console.warn("Failed to fetch actions:", error);
                if (isFirstLoad.current) {
                    setLoading(false);
                    isFirstLoad.current = false;
                }
            }
        };

        // Reset on database change
        setActions([]);
        isFirstLoad.current = true;
        setLoading(true);

        fetchActions();
        const intervalID = setInterval(fetchActions, interval);

        return () => clearInterval(intervalID);
    }, [interval, selectedDatabaseId]);

    return { actions, loading };
}