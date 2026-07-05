/**
 * 事件管理类型定义
 */

import type { IncidentPriority, IncidentSeverity, IncidentStatus } from '@/constants/incident';

// 事件实体接口
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: IncidentStatus;
  priority: IncidentPriority;
  severity: IncidentSeverity;
  incidentNumber?: string;
  reporterId?: number;
  assigneeId?: number;
  configurationItemId?: number;
  category: string;
  subcategory: string;
  impactAnalysis?: Record<string, any>;
  rootCause?: Record<string, any>;
  resolutionSteps?: Record<string, any>[];
  problemId?: number; // 关联的问题记录ID
  /** 版本号（乐观锁） */
  version?: number;
  detectedAt?: string;
  resolvedAt?: string;
  closedAt?: string;
  escalatedAt?: string;
  escalationLevel?: number;
  isAutomated?: boolean;
  source: string;
  metadata?: Record<string, any>;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
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
  /** 版本号（用于乐观锁冲突检测） */
  version?: number;
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
  incidents?: Incident[];
  items: Incident[];
  total: number;
  page?: number;
  pageSize?: number;
}
