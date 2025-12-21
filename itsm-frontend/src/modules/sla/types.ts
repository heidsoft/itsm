/**
 * SLA 类型定义
 */

export interface SLADefinition {
    id: number;
    name: string;
    description: string;
    service_type: string;
    priority: string;
    response_time: number;
    resolution_time: number;
    business_hours: Record<string, any>;
    escalation_rules: Record<string, any>;
    conditions: Record<string, any>;
    is_active: boolean;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

export interface SLAViolation {
    id: number;
    ticket_id: number;
    sla_definition_id: number;
    violation_type: string;
    violation_time: string;
    description: string;
    severity: string;
    is_resolved: boolean;
    resolved_at?: string;
    resolution_notes: string;
}

export interface SLAAlertRule {
    id: number;
    sla_definition_id: number;
    name: string;
    threshold_percentage: number;
    alert_level: string;
    notification_channels: string[];
    is_active: boolean;
}

export interface SLADefinitionListResponse {
    items: SLADefinition[];
    total: number;
    page: number;
    size: number;
}
