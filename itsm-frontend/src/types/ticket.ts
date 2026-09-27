/**
 * 工单相关类型定义
 *
 * ⚠️ 契约类型（Ticket / TicketStatus / TicketPriority / TicketType / TicketSource /
 * CreateTicketRequest / UpdateTicketRequest / TicketListResponse / TicketStatsResponse）
 * 的唯一来源是 `@/lib/api/ticket-api`（与后端 dto/ticket_dto.go 逐字段对齐）。
 * 本文件只保留前端视图/交互模型（筛选器、活动流、模板、批量操作等），
 * 禁止在此另起契约定义，也不得为契约同名类型再起别名（例如 TicketStatsResponse as TicketStats）。
 */

import { TicketStatus, TicketPriority } from '@/lib/api/ticket-api';
import type { Ticket, TicketListResponse, TicketType, TicketSource, CreateTicketRequest, UpdateTicketRequest, TicketStatsResponse, UserBasicInfo } from '@/lib/api/ticket-api';

export { TicketStatus, TicketPriority };
export type {
  Ticket,
  TicketListResponse,
  TicketType,
  TicketSource,
  CreateTicketRequest,
  UpdateTicketRequest,
  TicketStatsResponse,
  UserBasicInfo,
};

export interface TicketCategory {
  id: number;
  name: string;
  description?: string;
  parentId?: number;
  isActive: boolean;
}

/** 工单关联人员的视图模型；接口响应用 UserBasicInfo（@/lib/api/ticket-api） */
export interface TicketUser {
  id: number;
  username: string;
  name: string;
  email: string;
  role: string;
}

export interface TicketComment {
  id: number;
  ticketId: number;
  content: string;
  author: TicketUser;
  authorId?: number;
  isInternal: boolean;
  createdAt: string;
  updatedAt?: string;
  attachments?: TicketAttachment[];
}

export interface TicketAttachment {
  id: number;
  ticketId: number;
  filename: string;
  originalName: string;
  fileSize: number;
  mimeType: string;
  url: string;
  uploadedBy: TicketUser;
  createdAt: string;
}

export interface TicketSLA {
  id: number;
  name: string;
  responseTime: number; // 响应时间（分钟）
  resolutionTime: number; // 解决时间（分钟）
  priority: TicketPriority;
  isActive: boolean;
}

export interface TicketFilters {
  status?: TicketStatus[];
  priority?: TicketPriority[];
  type?: TicketType[];
  source?: TicketSource[];
  categoryId?: number[];
  assigneeId?: number[];
  requesterId?: number[];
  isMajorIncident?: boolean;
  slaStatus?: ('on_track' | 'at_risk' | 'breached')[];
  tags?: string[];
  dateRange?: {
    field: 'created' | 'updated' | 'due' | 'resolved' | 'closed';
    start: string;
    end: string;
  };
  search?: string;
}

export interface TicketSortOptions {
  field:
    | 'id'
    | 'ticketNumber'
    | 'title'
    | 'priority'
    | 'status'
    | 'createdAt'
    | 'updatedAt'
    | 'dueDate';
  order: 'asc' | 'desc';
}

export interface TicketStats {
  total: number;
  byStatus: Record<TicketStatus, number>;
  byPriority: Record<TicketPriority, number>;
  byType: Record<TicketType, number>;
  overdue: number;
  slaBreached: number;
  avgResolutionTime: number; // 小时
  avgResponseTime: number; // 小时
  satisfactionScore: number;
}

export interface TicketActivity {
  id: number;
  ticketId: number;
  type:
    | 'created'
    | 'updated'
    | 'assigned'
    | 'status_changed'
    | 'commented'
    | 'escalated'
    | 'resolved'
    | 'closed';
  description: string;
  actor: TicketUser;
  timestamp: string;
  changes?: Record<string, { from: unknown; to: unknown }>;
  metadata?: Record<string, unknown>;
}

// 工单模板
export interface TicketTemplate {
  id: number;
  name: string;
  description: string;
  category: TicketCategory;
  type: TicketType;
  priority: TicketPriority;
  titleTemplate: string;
  descriptionTemplate: string;
  customFields: Array<{
    name: string;
    type: 'text' | 'textarea' | 'select' | 'multiselect' | 'date' | 'number' | 'boolean';
    label: string;
    required: boolean;
    options?: string[];
    defaultValue?: unknown;
  }>;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// 工单批量操作
export interface TicketBatchOperation {
  action: 'assign' | 'update_status' | 'update_priority' | 'add_tags' | 'remove_tags' | 'delete';
  ticketIds: number[];
  data?: {
    assigneeId?: number;
    status?: TicketStatus;
    priority?: TicketPriority;
    tags?: string[];
    comment?: string;
  };
}

export interface TicketBatchResult {
  success: number;
  failed: number;
  errors: Array<{
    ticketId: number;
    error: string;
  }>;
}
