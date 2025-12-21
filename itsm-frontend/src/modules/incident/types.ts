/**
 * 事件管理类型定义
 */

import { IncidentPriority, IncidentSeverity, IncidentStatus } from "./constants";

// 事件实体接口
export interface Incident {
    id: number;
    title: string;
    description: string;
    status: IncidentStatus;
    priority: IncidentPriority;
    severity: IncidentSeverity;
    incident_number: string;
    reporter_id: number;
    assignee_id?: number;
    configuration_item_id?: number;
    category: string;
    subcategory: string;
    impact_analysis?: Record<string, any>; // Map<string, interface{}>
    root_cause?: Record<string, any>;
    resolution_steps?: Record<string, any>[];
    detected_at: string; // Time string
    resolved_at?: string;
    closed_at?: string;
    escalated_at?: string;
    escalation_level: number;
    is_automated: boolean;
    source: string;
    metadata?: Record<string, any>;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

// 事件活动记录
export interface IncidentEvent {
    id: number;
    incident_id: number;
    event_type: string;
    event_name: string;
    description: string;
    status: string;
    severity: string;
    data?: Record<string, any>;
    occurred_at: string;
    user_id?: number;
    source: string;
    created_at: string;
}

// 创建事件请求
export interface CreateIncidentRequest {
    title: string;
    description?: string;
    priority?: IncidentPriority;
    severity?: IncidentSeverity;
    category?: string;
    subcategory?: string;
    assignee_id?: number;
    configuration_item_id?: number;
    impact_analysis?: Record<string, any>;
    source?: string;
    detected_at?: string;
    metadata?: Record<string, any>;
}

// 更新事件请求
export interface UpdateIncidentRequest {
    title?: string;
    description?: string;
    status?: IncidentStatus;
    priority?: IncidentPriority;
    severity?: IncidentSeverity;
    category?: string;
    subcategory?: string;
    assignee_id?: number;
    impact_analysis?: Record<string, any>;
    root_cause?: Record<string, any>;
    resolution_steps?: Record<string, any>[];
    metadata?: Record<string, any>;
}

// 列表查询参数
export interface IncidentQuery {
    page?: number;
    size?: number;
    status?: string;
    priority?: string;
    keyword?: string;
    scope?: 'all' | 'me'; // all: 租户内所有, me: 分配给我的
}

// 列表响应
export interface IncidentListResponse {
    items: Incident[];
    total: number;
}
