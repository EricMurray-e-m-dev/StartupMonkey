'use client';

import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from 'react';

interface Database {
    database_id: string;
    database_name: string;
    database_type: string;
    connection_string: string;
    host: string;
    port: number;
    enabled: boolean;
    health_status?: string;
    health_score?: number;
}

interface DatabaseContextType {
    databases: Database[];
    selectedDatabaseId: string | null;
    selectedDatabase: Database | null;
    setSelectedDatabaseId: (id: string | null) => void;
    refreshDatabases: () => Promise<void>;
    loading: boolean;
    isAllSelected: boolean;
}

const DatabaseContext = createContext<DatabaseContextType>({
    databases: [],
    selectedDatabaseId: null,
    selectedDatabase: null,
    setSelectedDatabaseId: () => {},
    refreshDatabases: async () => {},
    loading: true,
    isAllSelected: true,
});

export const useDatabase = () => useContext(DatabaseContext);

const STORAGE_KEY = 'startupmonkey_selected_database';
export const ALL_DATABASES = 'all';

export default function DatabaseProvider({ children }: { children: ReactNode }) {
    const [databases, setDatabases] = useState<Database[]>([]);
    const [selectedDatabaseId, setSelectedDatabaseIdState] = useState<string | null>(ALL_DATABASES);
    const [loading, setLoading] = useState(true);

    const fetchDatabases = useCallback(async () => {
        try {
            const response = await fetch('/api/databases?enabled_only=true');
            if (response.ok) {
                const data = await response.json();
                setDatabases(data);
                return data;
            }
        } catch (error) {
            console.error('Failed to fetch databases:', error);
        }
        return [];
    }, []);

    const setSelectedDatabaseId = useCallback((id: string | null) => {
        setSelectedDatabaseIdState(id);
        if (id) {
            localStorage.setItem(STORAGE_KEY, id);
        } else {
            localStorage.removeItem(STORAGE_KEY);
        }
    }, []);

    // Initial load
    useEffect(() => {
        const init = async () => {
            const dbs = await fetchDatabases();
            
            // Restore selection from localStorage
            const savedId = localStorage.getItem(STORAGE_KEY);
            
            if (savedId === ALL_DATABASES) {
                // "All" was previously selected
                setSelectedDatabaseIdState(ALL_DATABASES);
            } else if (savedId && dbs.some((db: Database) => db.database_id === savedId)) {
                // Saved selection still valid
                setSelectedDatabaseIdState(savedId);
            } else {
                // Default to "All Databases"
                setSelectedDatabaseIdState(ALL_DATABASES);
                localStorage.setItem(STORAGE_KEY, ALL_DATABASES);
            }
            
            setLoading(false);
        };

        init();
    }, [fetchDatabases]);

    // Refresh periodically
    useEffect(() => {
        const interval = setInterval(fetchDatabases, 30000);
        return () => clearInterval(interval);
    }, [fetchDatabases]);

    const isAllSelected = selectedDatabaseId === ALL_DATABASES;
    const selectedDatabase = isAllSelected 
        ? null 
        : databases.find(db => db.database_id === selectedDatabaseId) || null;

    return (
        <DatabaseContext.Provider
            value={{
                databases,
                selectedDatabaseId,
                selectedDatabase,
                setSelectedDatabaseId,
                refreshDatabases: fetchDatabases,
                loading,
                isAllSelected,
            }}
        >
            {children}
        </DatabaseContext.Provider>
    );
}