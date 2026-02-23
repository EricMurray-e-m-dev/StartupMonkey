"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import { 
    Loader2, Save, RotateCcw, AlertCircle, CheckCircle, AlertTriangle, 
    Plus, Pencil, Trash2, Database, Circle 
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useDatabase } from "@/components/providers/DatabaseProvider";

interface DatabaseEntry {
    database_id: string;
    database_name: string;
    connection_string: string;
    database_type: string;
    host: string;
    port: number;
    enabled: boolean;
    health_status?: string;
    health_score?: number;
}

interface DetectionThresholds {
    connection_pool_critical: number;
    sequential_scan_threshold: number;
    sequential_scan_delta: number;
    p95_latency_ms: number;
    cache_hit_rate_threshold: number;
}

interface WebhookConfig {
    url: string;
    auth_header: string;
    enabled: boolean;
    events: string[];
}

interface SystemConfig {
    thresholds: DetectionThresholds | null;
    onboarding_complete: boolean;
    execution_mode?: string;
    webhook?: WebhookConfig | null;
}

const DEFAULT_THRESHOLDS: DetectionThresholds = {
    connection_pool_critical: 0.8,
    sequential_scan_threshold: 1000,
    sequential_scan_delta: 100,
    p95_latency_ms: 100,
    cache_hit_rate_threshold: 0.9,
};

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

const EMPTY_DATABASE: DatabaseEntry = {
    database_id: "",
    database_name: "",
    connection_string: "",
    database_type: "postgres",
    host: "",
    port: 5432,
    enabled: true,
};

export default function SettingsPage() {
    const { refreshDatabases } = useDatabase();
    
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [flushing, setFlushing] = useState(false);
    const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
    const [executionMode, setExecutionMode] = useState("autonomous");
    const [unavailableFeatures, setUnavailableFeatures] = useState<string[]>([]);

    // Database management
    const [databases, setDatabases] = useState<DatabaseEntry[]>([]);
    const [databaseDialogOpen, setDatabaseDialogOpen] = useState(false);
    const [editingDatabase, setEditingDatabase] = useState<DatabaseEntry | null>(null);
    const [databaseForm, setDatabaseForm] = useState<DatabaseEntry>(EMPTY_DATABASE);
    const [databaseSaving, setDatabaseSaving] = useState(false);
    const [databaseDeleting, setDatabaseDeleting] = useState<string | null>(null);

    const [thresholds, setThresholds] = useState<DetectionThresholds>(DEFAULT_THRESHOLDS);
    const [webhook, setWebhook] = useState<WebhookConfig>(DEFAULT_WEBHOOK);

    useEffect(() => {
        fetchConfig();
        fetchDatabases();
        fetchHealthStatus();
    }, []);

    const fetchDatabases = async () => {
        try {
            const response = await fetch("/api/databases");
            if (response.ok) {
                const data = await response.json();
                setDatabases(data);
            }
        } catch (error) {
            console.error("Failed to fetch databases:", error);
        }
    };

    const fetchConfig = async () => {
        try {
            const response = await fetch("/api/config");
            const config: SystemConfig = await response.json();

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

    // Database CRUD handlers
    const openAddDatabase = () => {
        setEditingDatabase(null);
        setDatabaseForm(EMPTY_DATABASE);
        setDatabaseDialogOpen(true);
    };

    const openEditDatabase = (db: DatabaseEntry) => {
        setEditingDatabase(db);
        setDatabaseForm({ ...db });
        setDatabaseDialogOpen(true);
    };

    const generateDatabaseId = (name: string) => {
        return name.toLowerCase().replace(/[^a-z0-9]/g, '_').replace(/_+/g, '_').replace(/^_|_$/g, '') || 'database';
    };

    const handleDatabaseNameChange = (name: string) => {
        setDatabaseForm(prev => ({
            ...prev,
            database_name: name,
            database_id: editingDatabase ? prev.database_id : generateDatabaseId(name),
        }));
    };

    const handleSaveDatabase = async () => {
        setDatabaseSaving(true);
        setMessage(null);

        try {
            if (editingDatabase) {
                // Update existing
                const response = await fetch(`/api/databases/${editingDatabase.database_id}`, {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        connection_string: databaseForm.connection_string,
                        database_name: databaseForm.database_name,
                        enabled: databaseForm.enabled,
                    }),
                });

                if (!response.ok) {
                    throw new Error("Failed to update database");
                }

                setMessage({ type: "success", text: "Database updated successfully." });
            } else {
                // Create new
                const response = await fetch("/api/databases", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(databaseForm),
                });

                if (!response.ok) {
                    throw new Error("Failed to add database");
                }

                setMessage({ type: "success", text: "Database added successfully." });
            }

            setDatabaseDialogOpen(false);
            fetchDatabases();
            refreshDatabases();
        } catch (error) {
            console.error("Failed to save database:", error);
            setMessage({ type: "error", text: error instanceof Error ? error.message : "Failed to save database" });
        } finally {
            setDatabaseSaving(false);
        }
    };

    const handleDeleteDatabase = async (databaseId: string) => {
        if (!confirm("Are you sure you want to remove this database? This will stop monitoring it.")) {
            return;
        }

        setDatabaseDeleting(databaseId);
        setMessage(null);

        try {
            const response = await fetch(`/api/databases/${databaseId}`, {
                method: "DELETE",
            });

            if (!response.ok) {
                throw new Error("Failed to remove database");
            }

            setMessage({ type: "success", text: "Database removed successfully." });
            fetchDatabases();
            refreshDatabases();
        } catch (error) {
            console.error("Failed to delete database:", error);
            setMessage({ type: "error", text: "Failed to remove database" });
        } finally {
            setDatabaseDeleting(null);
        }
    };

    const handleToggleEnabled = async (db: DatabaseEntry) => {
        try {
            const response = await fetch(`/api/databases/${db.database_id}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    connection_string: db.connection_string,
                    database_name: db.database_name,
                    enabled: !db.enabled,
                }),
            });

            if (!response.ok) {
                throw new Error("Failed to update database");
            }

            fetchDatabases();
            refreshDatabases();
        } catch (error) {
            console.error("Failed to toggle database:", error);
            setMessage({ type: "error", text: "Failed to update database" });
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setMessage(null);

        try {
            const config = {
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
                text: "Configuration saved successfully.",
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

    const toggleWebhookEvent = (eventId: string) => {
        setWebhook((prev) => ({
            ...prev,
            events: prev.events.includes(eventId)
                ? prev.events.filter((e) => e !== eventId)
                : [...prev.events, eventId],
        }));
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-full">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="space-y-6 max-w-4xl">
            <div>
                <h1 className="text-2xl font-bold">Settings</h1>
                <p className="text-muted-foreground">
                    Manage databases, thresholds, and system configuration
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

            {/* Database Management */}
            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <div>
                            <CardTitle>Databases</CardTitle>
                            <CardDescription>
                                Manage the databases that StartupMonkey monitors
                            </CardDescription>
                        </div>
                        <Button onClick={openAddDatabase} size="sm">
                            <Plus className="w-4 h-4 mr-2" />
                            Add Database
                        </Button>
                    </div>
                </CardHeader>
                <CardContent>
                    {databases.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground">
                            <Database className="w-12 h-12 mx-auto mb-4 opacity-50" />
                            <p>No databases configured</p>
                            <p className="text-sm">Add a database to start monitoring</p>
                        </div>
                    ) : (
                        <div className="space-y-3">
                            {databases.map((db) => (
                                <div
                                    key={db.database_id}
                                    className="flex items-center justify-between p-4 border rounded-lg"
                                >
                                    <div className="flex items-center gap-4">
                                        <div className="flex items-center gap-2">
                                            <HealthIndicator status={db.health_status} />
                                            <div>
                                                <p className="font-medium">{db.database_name}</p>
                                                <p className="text-sm text-muted-foreground">
                                                    {db.database_type} • {db.host}:{db.port}
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-4">
                                        <div className="flex items-center gap-2">
                                            <Label htmlFor={`enabled-${db.database_id}`} className="text-sm text-muted-foreground">
                                                {db.enabled ? "Enabled" : "Disabled"}
                                            </Label>
                                            <Switch
                                                id={`enabled-${db.database_id}`}
                                                checked={db.enabled}
                                                onCheckedChange={() => handleToggleEnabled(db)}
                                            />
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => openEditDatabase(db)}
                                        >
                                            <Pencil className="w-4 h-4" />
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => handleDeleteDatabase(db.database_id)}
                                            disabled={databaseDeleting === db.database_id}
                                        >
                                            {databaseDeleting === db.database_id ? (
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Trash2 className="w-4 h-4 text-destructive" />
                                            )}
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Database Dialog */}
            <Dialog open={databaseDialogOpen} onOpenChange={setDatabaseDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>
                            {editingDatabase ? "Edit Database" : "Add Database"}
                        </DialogTitle>
                        <DialogDescription>
                            {editingDatabase
                                ? "Update the database connection details."
                                : "Add a new database to monitor."}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="dialog-db-name">Database Name</Label>
                            <Input
                                id="dialog-db-name"
                                placeholder="My Production Database"
                                value={databaseForm.database_name}
                                onChange={(e) => handleDatabaseNameChange(e.target.value)}
                            />
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="dialog-db-type">Database Type</Label>
                            <Select
                                value={databaseForm.database_type}
                                onValueChange={(value) =>
                                    setDatabaseForm((prev) => ({ ...prev, database_type: value }))
                                }
                                disabled={!!editingDatabase}
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

                        <div className="space-y-2">
                            <Label htmlFor="dialog-conn-string">Connection String</Label>
                            <Input
                                id="dialog-conn-string"
                                type="password"
                                placeholder="postgresql://user:password@host:5432/database"
                                value={databaseForm.connection_string}
                                onChange={(e) =>
                                    setDatabaseForm((prev) => ({ ...prev, connection_string: e.target.value }))
                                }
                            />
                        </div>

                        <div className="flex items-center gap-2">
                            <Switch
                                id="dialog-enabled"
                                checked={databaseForm.enabled}
                                onCheckedChange={(checked: boolean) =>
                                    setDatabaseForm((prev) => ({ ...prev, enabled: checked }))
                                }
                            />
                            <Label htmlFor="dialog-enabled">Enable monitoring</Label>
                        </div>
                    </div>

                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDatabaseDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleSaveDatabase}
                            disabled={databaseSaving || !databaseForm.database_name || !databaseForm.connection_string}
                        >
                            {databaseSaving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                            {editingDatabase ? "Update" : "Add"}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

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
                        <Select value={executionMode} onValueChange={setExecutionMode}>
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
                        Configure when detections should trigger (applies to all databases)
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
                                Delete all detections, actions, and metrics history
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

function HealthIndicator({ status }: { status?: string }) {
    const color = status === 'healthy' 
        ? 'text-green-500' 
        : status === 'degraded' 
            ? 'text-yellow-500' 
            : 'text-muted-foreground';
    
    return <Circle className={`h-3 w-3 fill-current ${color}`} />;
}