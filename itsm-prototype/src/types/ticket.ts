/**
 * 工单相关类型定义
 */

export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent' | 'critical';
export type TicketStatus = 'new' | 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed' | 'cancelled';
export type TicketSource = 'web' | 'email' | 'phone' | 'chat' | 'api' | 'mobile';
export type TicketType = 'incident' | 'request' | 'problem' | 'change' | 'task';

export interface TicketCategory {
  id: number;
  name: string;
  description?: string;
  parentId?: number;
  isActive: boolean;
}

export interface TicketUser {
  id: number;
  username: string;
  fullName: string;
  email: string;
  avatar?: string;
  department?: string;
  role: string;
}

export interface TicketComment {
  id: number;
  ticketId: number;
  content: string;
  author: TicketUser;
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

export interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  source: TicketSource;
  type: TicketType;
  category?: TicketCategory;
  requester: TicketUser;
  assignee?: TicketUser;
  
  // 时间相关
  createdAt: string;
  updatedAt: string;
  dueDate?: string;
  resolvedAt?: string;
  closedAt?: string;
  
  // SLA相关
  sla?: TicketSLA;
  slaStatus?: 'on_track' | 'at_risk' | 'breached';
  responseDeadline?: string;
  resolutionDeadline?: string;
  
  // 解决方案
  resolution?: string;
  resolutionCategory?: string;
  
  // 满意度
  satisfactionRating?: number;
  satisfactionComment?: string;
  
  // 升级相关
  escalationLevel?: number;
  escalationReason?: string;
  
  // 关联数据
  comments?: TicketComment[];
  attachments?: TicketAttachment[];
  relatedTickets?: number[];
  
  // 自定义字段
  customFields?: Record<string, unknown>;
  
  // 元数据
  tenantId: number;
  isMajorIncident: boolean;
  tags?: string[];
}

export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: TicketPriority;
  type: TicketType;
  source?: TicketSource;
  categoryId?: number;
  assigneeId?: number;
  dueDate?: string;
  customFields?: Record<string, unknown>;
  tags?: string[];
  attachments?: File[];
}

export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  priority?: TicketPriority;
  status?: TicketStatus;
  categoryId?: number;
  assigneeId?: number;
  dueDate?: string;
  customFields?: Record<string, unknown>;
  tags?: string[];
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
  field: 'id' | 'ticketNumber' | 'title' | 'priority' | 'status' | 'createdAt' | 'updatedAt' | 'dueDate';
  order: 'asc' | 'desc';
}

export interface TicketListResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
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
  type: 'created' | 'updated' | 'assigned' | 'status_changed' | 'commented' | 'escalated' | 'resolved' | 'closed';
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