'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Loader2, Database, Settings, CheckCircle, AlertCircle } from 'lucide-react';

interface OnboardingWizardProps {
    onComplete: () => void;
}

interface DatabaseConfig {
    id: string;
    name: string;
    connection_string: string;
    type: string;
    enabled: boolean;
}

interface DetectionThresholds {
    connection_pool_critical: number;
    sequential_scan_threshold: number;
    sequential_scan_delta: number;
    p95_latency_ms: number;
    cache_hit_rate_threshold: number;
}

interface SystemConfig {
    database: DatabaseConfig;
    thresholds: DetectionThresholds;
    onboarding_complete: boolean;
}

const DEFAULT_THRESHOLDS: DetectionThresholds = {
    connection_pool_critical: 0.8,
    sequential_scan_threshold: 1000,
    sequential_scan_delta: 100,
    p95_latency_ms: 100,
    cache_hit_rate_threshold: 0.9,
};

export default function OnboardingWizard({ onComplete }: OnboardingWizardProps) {
    const [step, setStep] = useState(1);
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<'success' | 'error' | null>(null);
    const [error, setError] = useState<string | null>(null);

    const [database, setDatabase] = useState<DatabaseConfig>({
        id: '',
        name: '',
        connection_string: '',
        type: 'postgres',
        enabled: true,
    });

    const [thresholds, setThresholds] = useState<DetectionThresholds>(DEFAULT_THRESHOLDS);

    const generateDatabaseId = (name: string) => {
        return name.toLowerCase().replace(/[^a-z0-9]/g, '_').replace(/_+/g, '_').replace(/^_|_$/g, '') || 'database';
    };

    const handleNameChange = (name: string) => {
        setDatabase(prev => ({
            ...prev,
            name,
            id: generateDatabaseId(name),
        }));
    };

    const handleTestConnection = async () => {
        setTesting(true);
        setTestResult(null);
        setError(null);

        // For now, we just validate the format
        // Real connection test would require Collector to attempt connection
        try {
            const connStr = database.connection_string;
            
            if (!connStr) {
                throw new Error('Connection string is required');
            }

            if (!connStr.startsWith('postgres://') && !connStr.startsWith('postgresql://')) {
                throw new Error('Connection string must start with postgres:// or postgresql://');
            }

            // Basic format validation
            const hasHost = connStr.includes('@') && connStr.includes('/');
            if (!hasHost) {
                throw new Error('Invalid connection string format');
            }

            // Simulate brief delay
            await new Promise(resolve => setTimeout(resolve, 500));
            
            setTestResult('success');
        } catch (err) {
            setTestResult('error');
            setError(err instanceof Error ? err.message : 'Connection test failed');
        } finally {
            setTesting(false);
        }
    };

    const handleSaveConfig = async () => {
        setSaving(true);
        setError(null);

        try {
            const config: SystemConfig = {
                database,
                thresholds,
                onboarding_complete: true,
            };

            const response = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config),
            });

            if (!response.ok) {
                throw new Error('Failed to save configuration');
            }

            onComplete();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save configuration');
        } finally {
            setSaving(false);
        }
    };

    const canProceedStep1 = database.name && database.connection_string && testResult === 'success';

    return (
        <div className="min-h-screen bg-background flex items-center justify-center p-4">
            <Card className="w-full max-w-2xl">
                <CardHeader className="text-center">
                    <CardTitle className="text-2xl">Welcome to StartupMonkey</CardTitle>
                    <CardDescription>
                        Let&apos;s configure your database monitoring in a few simple steps
                    </CardDescription>
                    
                    {/* Progress indicator */}
                    <div className="flex justify-center gap-2 mt-4">
                        <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm ${step >= 1 ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}>
                            <Database className="w-4 h-4" />
                            Database
                        </div>
                        <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm ${step >= 2 ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}>
                            <Settings className="w-4 h-4" />
                            Thresholds
                        </div>
                    </div>
                </CardHeader>

                <CardContent>
                    {step === 1 && (
                        <div className="space-y-6">
                            <div className="space-y-2">
                                <Label htmlFor="db-name">Database Name</Label>
                                <Input
                                    id="db-name"
                                    placeholder="My Production Database"
                                    value={database.name}
                                    onChange={(e) => handleNameChange(e.target.value)}
                                />
                                <p className="text-sm text-muted-foreground">
                                    A friendly name to identify this database
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="db-type">Database Type</Label>
                                <Select
                                    value={database.type}
                                    onValueChange={(value) => setDatabase(prev => ({ ...prev, type: value }))}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="postgres">PostgreSQL</SelectItem>
                                        <SelectItem value="mysql" disabled>MySQL (coming soon)</SelectItem>
                                        <SelectItem value="sqlite" disabled>SQLite (coming soon)</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="conn-string">Connection String</Label>
                                <Input
                                    id="conn-string"
                                    type="password"
                                    placeholder="postgresql://user:password@host:5432/database"
                                    value={database.connection_string}
                                    onChange={(e) => setDatabase(prev => ({ ...prev, connection_string: e.target.value }))}
                                />
                                <p className="text-sm text-muted-foreground">
                                    Your PostgreSQL connection string
                                </p>
                            </div>

                            <div className="flex gap-2">
                                <Button
                                    variant="outline"
                                    onClick={handleTestConnection}
                                    disabled={!database.connection_string || testing}
                                >
                                    {testing && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                    Test Connection
                                </Button>
                                
                                {testResult === 'success' && (
                                    <div className="flex items-center text-green-600 gap-1">
                                        <CheckCircle className="w-4 h-4" />
                                        <span className="text-sm">Valid format</span>
                                    </div>
                                )}
                                
                                {testResult === 'error' && (
                                    <div className="flex items-center text-red-600 gap-1">
                                        <AlertCircle className="w-4 h-4" />
                                        <span className="text-sm">{error}</span>
                                    </div>
                                )}
                            </div>

                            <div className="flex justify-end pt-4">
                                <Button
                                    onClick={() => setStep(2)}
                                    disabled={!canProceedStep1}
                                >
                                    Continue
                                </Button>
                            </div>
                        </div>
                    )}

                    {step === 2 && (
                        <div className="space-y-6">
                            <p className="text-sm text-muted-foreground">
                                These defaults work for most applications. Adjust if needed.
                            </p>

                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <Label htmlFor="conn-pool">Connection Pool Alert (%)</Label>
                                    <Input
                                        id="conn-pool"
                                        type="number"
                                        min="0"
                                        max="100"
                                        value={Math.round(thresholds.connection_pool_critical * 100)}
                                        onChange={(e) => setThresholds(prev => ({
                                            ...prev,
                                            connection_pool_critical: parseInt(e.target.value) / 100
                                        }))}
                                    />
                                    <p className="text-xs text-muted-foreground">
                                        Alert when pool usage exceeds this %
                                    </p>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="seq-scan">Sequential Scan Threshold</Label>
                                    <Input
                                        id="seq-scan"
                                        type="number"
                                        min="0"
                                        value={thresholds.sequential_scan_threshold}
                                        onChange={(e) => setThresholds(prev => ({
                                            ...prev,
                                            sequential_scan_threshold: parseInt(e.target.value)
                                        }))}
                                    />
                                    <p className="text-xs text-muted-foreground">
                                        Minimum scans before alerting
                                    </p>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="latency">P95 Latency Alert (ms)</Label>
                                    <Input
                                        id="latency"
                                        type="number"
                                        min="0"
                                        value={thresholds.p95_latency_ms}
                                        onChange={(e) => setThresholds(prev => ({
                                            ...prev,
                                            p95_latency_ms: parseInt(e.target.value)
                                        }))}
                                    />
                                    <p className="text-xs text-muted-foreground">
                                        Alert when P95 latency exceeds this
                                    </p>
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="cache-hit">Cache Hit Rate Alert (%)</Label>
                                    <Input
                                        id="cache-hit"
                                        type="number"
                                        min="0"
                                        max="100"
                                        value={Math.round(thresholds.cache_hit_rate_threshold * 100)}
                                        onChange={(e) => setThresholds(prev => ({
                                            ...prev,
                                            cache_hit_rate_threshold: parseInt(e.target.value) / 100
                                        }))}
                                    />
                                    <p className="text-xs text-muted-foreground">
                                        Alert when hit rate drops below this %
                                    </p>
                                </div>
                            </div>

                            {error && (
                                <div className="flex items-center text-red-600 gap-1">
                                    <AlertCircle className="w-4 h-4" />
                                    <span className="text-sm">{error}</span>
                                </div>
                            )}

                            <div className="flex justify-between pt-4">
                                <Button variant="outline" onClick={() => setStep(1)}>
                                    Back
                                </Button>
                                <Button onClick={handleSaveConfig} disabled={saving}>
                                    {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                    Start Monitoring
                                </Button>
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    );
}