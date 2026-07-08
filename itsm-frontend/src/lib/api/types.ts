/**
 * 统一API类型定义
 * 包含所有API共用的类型定义
 */

// ==================== 基础类型 ====================

/** API响应基础结构 */
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

/** 分页响应结构 */
export interface PaginationResponse<T = unknown> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/** 列表查询参数 */
export interface ListQueryParams {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  keyword?: string;
}

/** 批量操作请求 */
export interface BatchOperationRequest {
  ids: (string | number)[];
  data?: Record<string, unknown>;
}

/** 批量操作响应 */
export interface BatchOperationResponse<T = unknown> {
  success: T[];
  failed: Array<{
    id: string | number;
    error: string;
  }>;
}

// ==================== 基础用户类型 ====================

export interface UserBasicInfo {
  id: number;
  name: string;
  username?: string;
  email?: string;
  avatar?: string;
}

// ==================== 工单相关类型 ====================

/** 工单优先级 - 与 @/constants/taxonomy 保持一致 */
export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent' | 'critical';
export type TicketStatus = 'new' | 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed';
export type TicketType = 'incident' | 'problem' | 'change' | 'service_request';

export interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  description: string;
  priority: TicketPriority;
  status: TicketStatus;
  type?: TicketType;
  category?: string;
  categoryId?: number;
  tags?: string[];
  requesterId: number;
  assigneeId?: number;
  assignee?: UserBasicInfo;
  requester?: UserBasicInfo;
  resolution?: string;
  slaId?: number;
  slaInfo?: SLAInfo;
  createdAt: string;
  updatedAt: string;
  dueTime?: string;
  closedAt?: string;
  /** 版本号（用于乐观锁冲突检测） */
  version?: number;
}

// TicketListResponse 已迁移到 api-config.ts（扩展 BaseTicket 含租户字段）
// 引用: import type { TicketListResponse } from './api-config';

export interface TicketCreateRequest {
  title: string;
  description?: string;
  priority: TicketPriority;
  type?: TicketType;
  categoryId?: number;
  tags?: string[];
  assigneeId?: number;
  formFields?: Record<string, unknown>;
  attachments?: string[];
}

export interface TicketUpdateRequest {
  title?: string;
  description?: string;
  priority?: TicketPriority;
  status?: TicketStatus;
  categoryId?: number;
  tags?: string[];
  assigneeId?: number;
  resolution?: string;
}

export interface TicketAssignRequest {
  assigneeId: number;
  comment?: string;
}

export interface TicketWorkflowStep {
  id: number;
  stepName: string;
  stepOrder: number;
  status: string;
  assigneeId?: number;
  assignee?: UserBasicInfo;
  startedAt?: string;
  completedAt?: string;
  comments?: string;
}

export interface SLAInfo {
  slaName: string;
  responseTime: number;
  resolutionTime: number;
  dueTime: string;
  status: 'active' | 'completed' | 'breached';
}

// ==================== 事件相关类型 ====================

export type IncidentSeverity = 'critical' | 'high' | 'medium' | 'low';
export type IncidentStatus =
  | 'new'
  | 'investigating'
  | 'identified'
  | 'monitoring'
  | 'resolved'
  | 'closed';

export interface Incident {
  id: number;
  incidentNumber: string;
  title: string;
  description: string;
  severity: IncidentSeverity;
  status: IncidentStatus;
  category?: string;
  priority?: TicketPriority;
  reporterId: number;
  reporter?: UserBasicInfo;
  assigneeId?: number;
  assignee?: UserBasicInfo;
  assigneeName?: string;
  impactAnalysis?: Record<string, unknown>;
  impact?: string;
  detectedAt: string;
  resolvedAt?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 变更相关类型 ====================

export type ChangeType = 'normal' | 'standard' | 'emergency';
export type ChangeStatus =
  | 'draft'
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'in_progress'
  | 'completed'
  | 'rolled_back'
  | 'cancelled';
export type ChangePriority = 'low' | 'medium' | 'high' | 'critical';
export type ChangeRisk = 'low' | 'medium' | 'high' | 'critical';

export interface Change {
  id: number;
  changeNumber: string;
  title: string;
  description: string;
  type: ChangeType;
  status: ChangeStatus;
  priority: ChangePriority;
  riskLevel: ChangeRisk;
  justification?: string;
  implementationPlan?: string;
  rollbackPlan?: string;
  plannedStartDate?: string;
  plannedEndDate?: string;
  assigneeId?: number;
  assignee?: UserBasicInfo;
  affectedCis?: string[];
  relatedTickets?: string[];
  createdAt: string;
  updatedAt: string;
}

// ==================== 用户相关类型 ====================

export interface User extends UserBasicInfo {
  phone?: string;
  department?: string;
  role?: string;
  permissions?: string[];
  status: 'active' | 'inactive' | 'locked';
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// ==================== 附件相关类型 ====================

export interface Attachment {
  id: number;
  ticketId?: number;
  fileName: string;
  filePath: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  uploadedBy: number;
  uploader?: UserBasicInfo;
  createdAt: string;
}

// ==================== 评论相关类型 ====================

export interface Comment {
  id: number;
  ticketId: number;
  userId: number;
  user?: UserBasicInfo;
  content: string;
  isInternal: boolean;
  mentions: number[];
  attachments: number[];
  createdAt: string;
  updatedAt: string;
}

export interface CommentCreateRequest {
  content: string;
  isInternal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

// ==================== 服务请求相关类型 ====================

export interface ServiceRequest {
  id: number;
  requestNumber: string;
  catalogId: number;
  catalogName?: string;
  title: string;
  reason?: string;
  status: 'pending' | 'approved' | 'rejected' | 'in_progress' | 'completed' | 'cancelled';
  requesterId: number;
  requester?: UserBasicInfo;
  formData?: Record<string, unknown>;
  approvalStatus?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ServiceCatalog {
  id: number;
  name: string;
  description?: string;
  icon?: string;
  parentId?: number;
  order: number;
  isActive: boolean;
  createdAt: string;
}

// ==================== 知识库相关类型 ====================

export interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  summary?: string;
  categoryId?: number;
  categoryName?: string;
  tags?: string[];
  authorId: number;
  author?: UserBasicInfo;
  viewCount: number;
  helpfulCount: number;
  status: 'draft' | 'published' | 'archived';
  createdAt: string;
  updatedAt: string;
}

// ==================== 工作流相关类型 ====================

export interface WorkflowDefinition {
  id: number;
  name: string;
  description?: string;
  xmlContent: string;
  version: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowInstance {
  id: number;
  workflowId: number;
  workflow?: WorkflowDefinition;
  entityType: string;
  entityId: number;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  currentNodeId?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

// ==================== 通知相关类型 ====================

export interface Notification {
  id: number;
  userId: number;
  title: string;
  content: string;
  type: 'info' | 'warning' | 'success' | 'error';
  isRead: boolean;
  readAt?: string;
  data?: Record<string, unknown>;
  createdAt: string;
}

// ==================== 审批相关类型 ====================

export interface ApprovalRequest {
  id: number;
  entityType: string;
  entityId: number;
  entityTitle: string;
  requesterId: number;
  requester?: UserBasicInfo;
  status: 'pending' | 'approved' | 'rejected';
  currentStep: number;
  totalSteps: number;
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalActionRequest {
  action: 'approve' | 'reject' | 'delegate';
  comment?: string;
  delegateToUserId?: number;
}

// ==================== SLA相关类型 ====================

export interface SLADefinition {
  id: number;
  name: string;
  description?: string;
  responseTime: number;
  resolutionTime: number;
  priority: TicketPriority;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// ==================== 仪表盘相关类型 ====================

export interface DashboardOverview {
  kpiMetrics: KPIMetric[];
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData: SLAComplianceData[];
  satisfactionData: SatisfactionData[];
  recentActivities: RecentActivity[];
}

export interface KPIMetric {
  id: string;
  title: string;
  value: number;
  unit: string;
  color: string;
  trend: 'up' | 'down' | 'stable';
  change: number;
  changeType: 'increase' | 'decrease' | 'stable';
  description?: string;
}

export interface TicketTrendData {
  date: string;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
}

export interface IncidentDistributionData {
  category: string;
  count: number;
  color?: string;
}

export interface SLAComplianceData {
  service: string;
  target: number;
  actual: number;
}

export interface SatisfactionData {
  month: string;
  rating: number;
  responses: number;
}

export interface RecentActivity {
  id: string;
  type: string;
  title: string;
  description: string;
  user: string;
  timestamp: string;
  priority?: string;
  status?: string;
}

// ==================== 错误码定义 ====================

export enum ApiErrorCode {
  SUCCESS = 0,
  PARAM_ERROR = 1001,
  AUTH_FAILED = 2001,
  FORBIDDEN = 2003,
  NOT_FOUND = 4004,
  INTERNAL_ERROR = 5001,
}

// ==================== 请求选项 ====================

export interface RequestOptions {
  timeout?: number;
  retry?: boolean;
  retryCount?: number;
}

export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
}

export interface FileUploadOptions {
  file: File;
  onProgress?: (progress: UploadProgress) => void;
  additionalData?: Record<string, string>;
}

// ==================== API URL 和工具函数 ====================

export const API_URLS = {
  INCIDENTS: () => '/api/v1/incidents',
  INCIDENT: (id: number) => `/api/v1/incidents/${id}`,
};

export function normalizePaginationParams(
  params: Record<string, unknown>
): Record<string, unknown> {
  const normalized: Record<string, unknown> = {};
  if (params.page !== undefined) normalized.page = params.page;
  if (params.pageSize !== undefined) normalized.pageSize = params.pageSize;
  return normalized;
}

export function normalizeDateRangeParams(params: Record<string, unknown>): Record<string, unknown> {
  const normalized: Record<string, unknown> = {};
  if (params.dateFrom !== undefined) normalized.dateFrom = params.dateFrom;
  if (params.dateTo !== undefined) normalized.dateTo = params.dateTo;
  return normalized;
}

// 兼容性重导出
export type { UserBasicInfo as Permission } from './types';
