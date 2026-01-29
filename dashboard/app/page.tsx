'use client';

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Activity, AlertTriangle, Zap, Database, CheckCircle, XCircle, Clock, Eye, Settings } from "lucide-react";
import { useDetections } from "@/hooks/useDetections";
import { useActions } from "@/hooks/useActions";
import { useEffect, useState } from "react";
import Link from "next/link";

interface SystemConfig {
  database?: {
    id: string;
    name: string;
    type: string;
  };
  execution_mode?: string;
  onboarding_complete?: boolean;
}

export default function Home() {
  const { detections } = useDetections(5000);
  const { actions } = useActions(5000);
  const [config, setConfig] = useState<SystemConfig | null>(null);
  const [servicesHealth, setServicesHealth] = useState<Record<string, boolean>>({});

  // Fetch system config
  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch('/api/config');
        if (response.ok) {
          const data = await response.json();
          setConfig(data);
        }
      } catch (error) {
        console.error('Failed to fetch config:', error);
      }
    };

    fetchConfig();
    const interval = setInterval(fetchConfig, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  // Check service health
  useEffect(() => {
    const checkHealth = async () => {
      const services = [
        { name: 'collector', url: '/api/health' },
      ];

      const health: Record<string, boolean> = {};
      
      for (const service of services) {
        try {
          const response = await fetch(service.url);
          health[service.name] = response.ok;
        } catch {
          health[service.name] = false;
        }
      }

      // If we're getting detections/actions, services are likely working
      health['analyser'] = detections.length > 0 || health['collector'];
      health['executor'] = actions.length > 0 || health['collector'];

      setServicesHealth(health);
    };

    checkHealth();
    const interval = setInterval(checkHealth, 10000);
    return () => clearInterval(interval);
  }, [detections.length, actions.length]);

  // Calculate stats
  const activeDetections = detections;
  const criticalDetections = activeDetections.filter(d => d.severity === 'critical').length;
  const warningDetections = activeDetections.filter(d => d.severity === 'warning').length;

  const pendingApproval = actions.filter(a => a.status === 'pending_approval').length;
  const suggested = actions.filter(a => a.status === 'suggested').length;
  const executing = actions.filter(a => a.status === 'executing').length;
  const completed = actions.filter(a => a.status === 'completed').length;
  const failed = actions.filter(a => a.status === 'failed').length;

  const executionModeDisplay = {
    autonomous: { label: 'Autonomous', color: 'bg-green-500' },
    approval: { label: 'Approval', color: 'bg-orange-500' },
    observe: { label: 'Observe', color: 'bg-slate-500' },
  };

  const currentMode = executionModeDisplay[config?.execution_mode as keyof typeof executionModeDisplay] || executionModeDisplay.autonomous;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Overview</h1>
          <p className="text-muted-foreground">
            System status and recent activity
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Badge className={`${currentMode.color} text-white`}>
            {currentMode.label} Mode
          </Badge>
          <Link href="/settings">
            <Settings className="h-4 w-4 text-muted-foreground hover:text-foreground cursor-pointer" />
          </Link>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Database Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Database
            </CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {config?.database ? (
              <>
                <div className="text-lg font-bold truncate">{config.database.name}</div>
                <p className="text-xs text-muted-foreground">
                  {config.database.type} • {config.database.id}
                </p>
              </>
            ) : (
              <>
                <div className="text-lg font-bold text-muted-foreground">Not configured</div>
                <Link href="/settings" className="text-xs text-blue-500 hover:underline">
                  Configure database →
                </Link>
              </>
            )}
          </CardContent>
        </Card>

        {/* Active Detections */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Active Detections
            </CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeDetections.length}</div>
            <p className="text-xs text-muted-foreground">
              {criticalDetections > 0 && <span className="text-red-500">{criticalDetections} critical</span>}
              {criticalDetections > 0 && warningDetections > 0 && ', '}
              {warningDetections > 0 && <span className="text-yellow-500">{warningDetections} warning</span>}
              {criticalDetections === 0 && warningDetections === 0 && 'No active issues'}
            </p>
          </CardContent>
        </Card>

        {/* Actions */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Actions
            </CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{actions.length}</div>
            <p className="text-xs text-muted-foreground">
              {pendingApproval > 0 && <span className="text-orange-500">{pendingApproval} pending approval</span>}
              {pendingApproval > 0 && (suggested > 0 || executing > 0) && ', '}
              {suggested > 0 && <span className="text-slate-500">{suggested} suggested</span>}
              {suggested > 0 && executing > 0 && ', '}
              {executing > 0 && <span className="text-yellow-500">{executing} executing</span>}
              {pendingApproval === 0 && suggested === 0 && executing === 0 && `${completed} completed, ${failed} failed`}
            </p>
          </CardContent>
        </Card>

        {/* Services Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Services
            </CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex gap-2 flex-wrap">
              <ServiceBadge name="Collector" healthy={servicesHealth['collector']} />
              <ServiceBadge name="Analyser" healthy={servicesHealth['analyser']} />
              <ServiceBadge name="Executor" healthy={servicesHealth['executor']} />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Activity */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* Recent Detections */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-lg">Recent Detections</CardTitle>
              <Link href="/detections" className="text-xs text-blue-500 hover:underline">
                View all →
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {detections.length === 0 ? (
              <p className="text-sm text-muted-foreground">No detections yet</p>
            ) : (
              <div className="space-y-3">
                {detections.slice(0, 5).map((detection, i) => (
                  <div key={detection.id || i} className="flex items-start gap-3">
                    <AlertTriangle className={`h-4 w-4 mt-0.5 ${
                      detection.severity === 'critical' ? 'text-red-500' : 'text-yellow-500'
                    }`} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{detection.title}</p>
                      <p className="text-xs text-muted-foreground">{detection.detector_name}</p>
                    </div>
                    <Badge variant={detection.severity === 'critical' ? 'destructive' : 'secondary'} className="text-xs">
                      {detection.severity}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Actions */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-lg">Recent Actions</CardTitle>
              <Link href="/actions" className="text-xs text-blue-500 hover:underline">
                View all →
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {actions.length === 0 ? (
              <p className="text-sm text-muted-foreground">No actions yet</p>
            ) : (
              <div className="space-y-3">
                {actions.slice(0, 5).map((action, i) => (
                  <div key={action.action_id || i} className="flex items-start gap-3">
                    <ActionStatusIcon status={action.status} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {action.action_type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                      </p>
                      <p className="text-xs text-muted-foreground">{action.database_id}</p>
                    </div>
                    <Badge variant="secondary" className="text-xs">
                      {action.status.replace(/_/g, ' ')}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function ServiceBadge({ name, healthy }: { name: string; healthy?: boolean }) {
  if (healthy === undefined) {
    return <Badge variant="secondary" className="text-xs">{name}</Badge>;
  }
  
  return (
    <Badge variant={healthy ? "default" : "destructive"} className="text-xs">
      {healthy ? <CheckCircle className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
      {name}
    </Badge>
  );
}

function ActionStatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'completed':
      return <CheckCircle className="h-4 w-4 mt-0.5 text-green-500" />;
    case 'failed':
      return <XCircle className="h-4 w-4 mt-0.5 text-red-500" />;
    case 'executing':
      return <Zap className="h-4 w-4 mt-0.5 text-yellow-500" />;
    case 'pending_approval':
      return <Clock className="h-4 w-4 mt-0.5 text-orange-500" />;
    case 'suggested':
      return <Eye className="h-4 w-4 mt-0.5 text-slate-500" />;
    default:
      return <Clock className="h-4 w-4 mt-0.5 text-blue-500" />;
  }
}