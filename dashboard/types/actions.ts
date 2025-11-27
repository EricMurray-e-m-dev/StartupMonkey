export type ActionStatus = 'queued' | 'executing' | 'completed' | 'failed' | 'rolled_back';

export interface ActionResult {
    action_id: string;
    detection_id: string;
    action_type: string;
    database_id: string;
    status: ActionStatus;
    message: string;
    created_at: string;
    started?: string;
    completed?: string;
    execution_time_ms: number;
    changes?: Record<string, string | number | boolean>;
    error?: string;
    can_rollback: boolean;
    rolledback: boolean;
    rollback_error?: string;
}