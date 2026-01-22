'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import OnboardingWizard from '@/components/onboarding/OnboardingWizard';
import { Loader2 } from 'lucide-react';

interface OnboardingContextType {
    isConfigured: boolean;
    refreshStatus: () => Promise<void>;
}

const OnboardingContext = createContext<OnboardingContextType>({
    isConfigured: false,
    refreshStatus: async () => {},
});

export const useOnboarding = () => useContext(OnboardingContext);

export default function OnboardingProvider({ children }: { children: ReactNode }) {
    const [loading, setLoading] = useState(true);
    const [isConfigured, setIsConfigured] = useState(false);

    const checkStatus = async () => {
        try {
            const response = await fetch('/api/status');
            const status = await response.json();
            setIsConfigured(status.onboarding_complete === true);
        } catch (error) {
            console.error('Failed to check onboarding status:', error);
            // On error, assume not configured
            setIsConfigured(false);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        checkStatus();
    }, []);

    const handleOnboardingComplete = () => {
        setIsConfigured(true);
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!isConfigured) {
        return <OnboardingWizard onComplete={handleOnboardingComplete} />;
    }

    return (
        <OnboardingContext.Provider value={{ isConfigured, refreshStatus: checkStatus }}>
            {children}
        </OnboardingContext.Provider>
    );
}