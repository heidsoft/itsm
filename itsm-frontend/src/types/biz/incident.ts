/**
 * 事件管理类型定义
 */

import { IncidentPriority, IncidentSeverity, IncidentStatus } from '@/constants/incident';

// 事件实体接口
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: IncidentStatus;
  priority: IncidentPriority;
  severity: IncidentSeverity;
  // 同时支持 camelCase 和 snake_case
  incidentNumber?: string;
  incident_number?: string;
  reporterId?: number;
  reporter_id?: number;
  assigneeId?: number;
  assignee_id?: number;
  configurationItemId?: number;
  configuration_item_id?: number;
  category: string;
  subcategory: string;
  impactAnalysis?: Record<string, any>; // Map<string, interface{}>
  impact_analysis?: Record<string, any>;
  rootCause?: Record<string, any>;
  root_cause?: Record<string, any>;
  resolutionSteps?: Record<string, any>[];
  resolution_steps?: Record<string, any>[];
  detectedAt?: string; // Time string
  detected_at?: string;
  resolvedAt?: string;
  resolved_at?: string;
  closedAt?: string;
  closed_at?: string;
  escalatedAt?: string;
  escalated_at?: string;
  escalationLevel?: number;
  escalation_level?: number;
  isAutomated?: boolean;
  is_automated?: boolean;
  source: string;
  metadata?: Record<string, any>;
  tenantId?: number;
  tenant_id?: number;
  createdAt?: string;
  created_at?: string;
  updatedAt?: string;
  updated_at?: string;
}

// 事件活动记录
export interface IncidentEvent {
  id: number;
  incidentId: number;
  eventType: string;
  eventName: string;
  description: string;
  status: string;
  severity: string;
  data?: Record<string, any>;
  occurredAt: string;
  userId?: number;
  source: string;
  createdAt: string;
}

// 创建事件请求
export interface CreateIncidentRequest {
  title: string;
  description?: string;
  priority?: IncidentPriority;
  severity?: IncidentSeverity;
  category?: string;
  subcategory?: string;
  assigneeId?: number;
  configurationItemId?: number;
  impactAnalysis?: Record<string, any>;
  source?: string;
  detectedAt?: string;
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
  assigneeId?: number;
  impactAnalysis?: Record<string, any>;
  rootCause?: Record<string, any>;
  resolutionSteps?: Record<string, any>[];
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
