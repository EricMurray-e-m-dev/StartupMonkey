"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2, Save, RotateCcw, AlertCircle, CheckCircle, AlertTriangle } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

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
    database: DatabaseConfig | null;
    thresholds: DetectionThresholds | null;
    onboarding_complete: boolean;
    execution_mode?: string;
}

const DEFAULT_THRESHOLDS: DetectionThresholds = {
    connection_pool_critical: 0.8,
    sequential_scan_threshold: 1000,
    sequential_scan_delta: 100,
    p95_latency_ms: 100,
    cache_hit_rate_threshold: 0.9,
};

interface WebhookConfig {
    url: string;
    auth_header: string;
    enabled: boolean;
    events: string[];
}

interface SystemConfig {
    database: DatabaseConfig | null;
    thresholds: DetectionThresholds | null;
    onboarding_complete: boolean;
    execution_mode?: string;
    webhook?: WebhookConfig | null;
}

const DEFAULT_WEBHOOK: WebhookConfig = {
    url: "",
    auth_header: "",
    enabled: false,
    events: ["detection.created", "action.completed", "action.failed"],
};

const WEBHOOK_EVENTS = [
    { id: "detection.created", label: "Detection Created" },
    { id: "action.queued", label: "Action Queued" },
    { id: "action.completed", label: "Action Completed" },
    { id: "action.failed", label: "Action Failed" },
    { id: "action.rolledback", label: "Action Rolled Back" },
];

export default function SettingsPage() {
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [flushing, setFlushing] = useState(false);
    const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
    const [executionMode, setExecutionMode] = useState("autonomous");
    const [unavailableFeatures, setUnavailableFeatures] = useState<string[]>([]);

    const [database, setDatabase] = useState<DatabaseConfig>({
        id: "",
        name: "",
        connection_string: "",
        type: "postgres",
        enabled: true,
    });

    const [thresholds, setThresholds] = useState<DetectionThresholds>(DEFAULT_THRESHOLDS);
    const [webhook, setWebhook] = useState<WebhookConfig>(DEFAULT_WEBHOOK);

    useEffect(() => {
        fetchConfig();
        fetchHealthStatus();
    }, []);

    const fetchConfig = async () => {
    try {
        const response = await fetch("/api/config");
        const config: SystemConfig = await response.json();

        if (config.database) {
            setDatabase(config.database);
        }

        if (config.thresholds) {
            setThresholds(config.thresholds);
        }

        if (config.execution_mode) {
            setExecutionMode(config.execution_mode);
        }

        if (config.webhook) {
            setWebhook(config.webhook);
        }
    } catch (error) {
        console.error("Failed to fetch config:", error);
        setMessage({ type: "error", text: "Failed to load configuration" });
    } finally {
        setLoading(false);
    }
};

    const fetchHealthStatus = async () => {
        try {
            const response = await fetch("/api/health");
            if (response.ok) {
                const health = await response.json();
                if (health.unavailable_features) {
                    setUnavailableFeatures(health.unavailable_features);
                }
            }
        } catch (error) {
            console.error("Failed to fetch health status:", error);
        }
    };

    const handleSave = async () => {
    setSaving(true);
    setMessage(null);

    try {
        const config = {
            database,
            thresholds,
            onboarding_complete: true,
            execution_mode: executionMode,
            webhook,
        };

        const response = await fetch("/api/config", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(config),
        });

        if (!response.ok) {
            throw new Error("Failed to save configuration");
        }

        setMessage({
            type: "success",
            text: "Configuration saved. Restart Collector and Analyser to apply changes.",
        });
        } catch (error) {
            console.error("Failed to save config:", error);
            setMessage({ type: "error", text: "Failed to save configuration" });
        } finally {
            setSaving(false);
        }
    };

    const handleFlush = async () => {
        if (!confirm("This will delete ALL data including detections, actions, and metrics history. Are you sure?")) {
            return;
        }

        setFlushing(true);
        setMessage(null);

        try {
            const response = await fetch("/api/flush", {
                method: "POST",
            });

            if (!response.ok) {
                throw new Error("Failed to flush data");
            }

            setMessage({ type: "success", text: "All data has been flushed successfully." });
        } catch (error) {
            console.error("Failed to flush:", error);
            setMessage({ type: "error", text: "Failed to flush data" });
        } finally {
            setFlushing(false);
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-full">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    const toggleWebhookEvent = (eventId: string) => {
        setWebhook((prev) => ({
            ...prev,
            events: prev.events.includes(eventId)
                ? prev.events.filter((e) => e !== eventId)
                : [...prev.events, eventId],
        }));
    };

    return (
        <div className="space-y-6 max-w-4xl">
            <div>
                <h1 className="text-2xl font-bold">Settings</h1>
                <p className="text-muted-foreground">
                    Configure your database connection and detection thresholds
                </p>
            </div>

            {message && (
                <Alert variant={message.type === "error" ? "destructive" : "default"}>
                    {message.type === "success" ? (
                        <CheckCircle className="h-4 w-4" />
                    ) : (
                        <AlertCircle className="h-4 w-4" />
                    )}
                    <AlertDescription>{message.text}</AlertDescription>
                </Alert>
            )}

            {/* Unavailable Features Warning */}
            {unavailableFeatures.length > 0 && (
                <Card className="border-amber-500/50 bg-amber-50 dark:bg-amber-950/20">
                    <CardHeader className="pb-2">
                        <CardTitle className="text-amber-700 dark:text-amber-500 flex items-center gap-2 text-base">
                            <AlertTriangle className="h-4 w-4" />
                            Limited Functionality
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <p className="text-sm text-amber-800 dark:text-amber-400 mb-2">
                            The following extensions could not be enabled:
                        </p>
                        <ul className="text-sm text-amber-700 dark:text-amber-500 list-disc list-inside space-y-1">
                            {unavailableFeatures.map((feature) => (
                                <li key={feature}>
                                    <code className="bg-amber-100 dark:bg-amber-900/50 px-1 rounded">{feature}</code>
                                    {feature === "pg_stat_statements" && (
                                        <span className="text-muted-foreground"> — Required for slow query analysis and index recommendations</span>
                                    )}
                                </li>
                            ))}
                        </ul>
                        <p className="text-xs text-muted-foreground mt-3">
                            Add to postgresql.conf: <code className="bg-muted px-1 rounded">shared_preload_libraries = &apos;pg_stat_statements&apos;</code> and restart PostgreSQL.
                        </p>
                    </CardContent>
                </Card>
            )}

            {/* Database Configuration */}
            <Card>
                <CardHeader>
                    <CardTitle>Database Connection</CardTitle>
                    <CardDescription>
                        Configure the database that StartupMonkey monitors
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <Label htmlFor="db-name">Database Name</Label>
                            <Input
                                id="db-name"
                                value={database.name}
                                onChange={(e) =>
                                    setDatabase((prev) => ({
                                        ...prev,
                                        name: e.target.value,
                                        id: e.target.value.toLowerCase().replace(/[^a-z0-9]/g, "_"),
                                    }))
                                }
                            />
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="db-type">Database Type</Label>
                            <Select
                                value={database.type}
                                onValueChange={(value) =>
                                    setDatabase((prev) => ({ ...prev, type: value }))
                                }
                            >
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="postgres">PostgreSQL</SelectItem>
                                    <SelectItem value="mysql" disabled>MySQL (coming soon)</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="conn-string">Connection String</Label>
                        <Input
                            id="conn-string"
                            type="password"
                            placeholder="postgresql://user:password@host:5432/database"
                            value={database.connection_string}
                            onChange={(e) =>
                                setDatabase((prev) => ({ ...prev, connection_string: e.target.value }))
                            }
                        />
                        <p className="text-xs text-muted-foreground">
                            Changes require Collector restart to take effect
                        </p>
                    </div>
                </CardContent>
            </Card>

            {/* Execution Mode */}
            <Card>
                <CardHeader>
                    <CardTitle>Execution Mode</CardTitle>
                    <CardDescription>
                        Control how StartupMonkey responds to detected issues
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="exec-mode">Mode</Label>
                        <Select
                            value={executionMode}
                            onValueChange={setExecutionMode}
                        >
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="autonomous">
                                    Autonomous - Execute actions automatically
                                </SelectItem>
                                <SelectItem value="approval">
                                    Approval - Wait for user approval before executing
                                </SelectItem>
                                <SelectItem value="observe">
                                    Observe - Detect issues only, no actions taken
                                </SelectItem>
                            </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">
                            {executionMode === "autonomous" && "Actions will be executed immediately when issues are detected."}
                            {executionMode === "approval" && "Actions will be queued and require your approval in the Actions page."}
                            {executionMode === "observe" && "Issues will be detected and logged, but no actions will be taken."}
                        </p>
                    </div>
                </CardContent>
            </Card>

            {/* Detection Thresholds */}
            <Card>
                <CardHeader>
                    <CardTitle>Detection Thresholds</CardTitle>
                    <CardDescription>
                        Configure when detections should trigger. Changes require Analyser restart.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-2 gap-6">
                        <div className="space-y-2">
                            <Label htmlFor="conn-pool">Connection Pool Alert (%)</Label>
                            <Input
                                id="conn-pool"
                                type="number"
                                min="0"
                                max="100"
                                value={Math.round(thresholds.connection_pool_critical * 100)}
                                onChange={(e) =>
                                    setThresholds((prev) => ({
                                        ...prev,
                                        connection_pool_critical: parseInt(e.target.value) / 100,
                                    }))
                                }
                            />
                            <p className="text-xs text-muted-foreground">
                                Alert when pool usage exceeds this percentage
                            </p>
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="seq-scan">Sequential Scan Threshold</Label>
                            <Input
                                id="seq-scan"
                                type="number"
                                min="0"
                                value={thresholds.sequential_scan_threshold}
                                onChange={(e) =>
                                    setThresholds((prev) => ({
                                        ...prev,
                                        sequential_scan_threshold: parseInt(e.target.value),
                                    }))
                                }
                            />
                            <p className="text-xs text-muted-foreground">
                                Minimum sequential scans before alerting
                            </p>
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="seq-delta">Sequential Scan Delta</Label>
                            <Input
                                id="seq-delta"
                                type="number"
                                min="0"
                                value={thresholds.sequential_scan_delta}
                                onChange={(e) =>
                                    setThresholds((prev) => ({
                                        ...prev,
                                        sequential_scan_delta: parseFloat(e.target.value),
                                    }))
                                }
                            />
                            <p className="text-xs text-muted-foreground">
                                Minimum increase between cycles to alert
                            </p>
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="latency">P95 Latency Alert (ms)</Label>
                            <Input
                                id="latency"
                                type="number"
                                min="0"
                                value={thresholds.p95_latency_ms}
                                onChange={(e) =>
                                    setThresholds((prev) => ({
                                        ...prev,
                                        p95_latency_ms: parseFloat(e.target.value),
                                    }))
                                }
                            />
                            <p className="text-xs text-muted-foreground">
                                Alert when P95 query latency exceeds this
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
                                onChange={(e) =>
                                    setThresholds((prev) => ({
                                        ...prev,
                                        cache_hit_rate_threshold: parseInt(e.target.value) / 100,
                                    }))
                                }
                            />
                            <p className="text-xs text-muted-foreground">
                                Alert when cache hit rate drops below this
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Webhook Configuration */}
            <Card>
                <CardHeader>
                    <CardTitle>Webhook Notifications</CardTitle>
                    <CardDescription>
                        Send HTTP notifications when events occur
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            id="webhook-enabled"
                            checked={webhook.enabled}
                            onChange={(e) =>
                                setWebhook((prev) => ({ ...prev, enabled: e.target.checked }))
                            }
                            className="h-4 w-4 rounded border-gray-300"
                        />
                        <Label htmlFor="webhook-enabled">Enable webhooks</Label>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="webhook-url">Webhook URL</Label>
                        <Input
                            id="webhook-url"
                            type="url"
                            placeholder="https://your-server.com/webhook"
                            value={webhook.url}
                            onChange={(e) =>
                                setWebhook((prev) => ({ ...prev, url: e.target.value }))
                            }
                            disabled={!webhook.enabled}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="webhook-auth">Authorization Header (optional)</Label>
                        <Input
                            id="webhook-auth"
                            type="password"
                            placeholder="Bearer your-token-here"
                            value={webhook.auth_header}
                            onChange={(e) =>
                                setWebhook((prev) => ({ ...prev, auth_header: e.target.value }))
                            }
                            disabled={!webhook.enabled}
                        />
                        <p className="text-xs text-muted-foreground">
                            Sent as the Authorization header with each request
                        </p>
                    </div>

                    <div className="space-y-2">
                        <Label>Events</Label>
                        <div className="grid grid-cols-2 gap-2">
                            {WEBHOOK_EVENTS.map((event) => (
                                <div key={event.id} className="flex items-center space-x-2">
                                    <input
                                        type="checkbox"
                                        id={`event-${event.id}`}
                                        checked={webhook.events.includes(event.id)}
                                        onChange={() => toggleWebhookEvent(event.id)}
                                        disabled={!webhook.enabled}
                                        className="h-4 w-4 rounded border-gray-300"
                                    />
                                    <Label
                                        htmlFor={`event-${event.id}`}
                                        className={!webhook.enabled ? "text-muted-foreground" : ""}
                                    >
                                        {event.label}
                                    </Label>
                                </div>
                            ))}
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Danger Zone */}
            <Card className="border-destructive/50">
                <CardHeader>
                    <CardTitle className="text-destructive">Danger Zone</CardTitle>
                    <CardDescription>
                        Irreversible actions that affect all data
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="font-medium">Flush All Data</p>
                            <p className="text-sm text-muted-foreground">
                                Delete all detections, actions, metrics history, and configuration
                            </p>
                        </div>
                        <Button
                            variant="destructive"
                            onClick={handleFlush}
                            disabled={flushing}
                        >
                            {flushing && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                            <RotateCcw className="w-4 h-4 mr-2" />
                            Flush All Data
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {/* Save Button */}
            <div className="flex justify-end">
                <Button onClick={handleSave} disabled={saving}>
                    {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                    <Save className="w-4 h-4 mr-2" />
                    Save Configuration
                </Button>
            </div>
        </div>
    );
}