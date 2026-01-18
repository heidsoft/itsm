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
  page_size: number;
  total_pages: number;
}

/** 列表查询参数 */
export interface ListQueryParams {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
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

// ==================== 工单相关类型 ====================

export type TicketPriority = 'low' | 'medium' | 'high' | 'critical';
export type TicketStatus = 'new' | 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed';
export type TicketType = 'incident' | 'problem' | 'change' | 'service_request';

export interface Ticket {
  id: number;
  ticket_number: string;
  title: string;
  description: string;
  priority: TicketPriority;
  status: TicketStatus;
  type: TicketType;
  category?: string;
  category_id?: number;
  tags?: string[];
  requester_id: number;
  assignee_id?: number;
  assignee?: UserBasicInfo;
  requester?: UserBasicInfo;
  resolution?: string;
  sla_id?: number;
  sla_info?: SLAInfo;
  created_at: string;
  updated_at: string;
  due_time?: string;
  closed_at?: string;
}

export interface TicketCreateRequest {
  title: string;
  description?: string;
  priority: TicketPriority;
  type?: TicketType;
  category_id?: number;
  tags?: string[];
  assignee_id?: number;
  form_fields?: Record<string, unknown>;
  attachments?: string[];
}

export interface TicketUpdateRequest {
  title?: string;
  description?: string;
  priority?: TicketPriority;
  status?: TicketStatus;
  category_id?: number;
  tags?: string[];
  assignee_id?: number;
  resolution?: string;
}

export interface TicketAssignRequest {
  assignee_id: number;
  comment?: string;
}

export interface TicketWorkflowStep {
  id: number;
  step_name: string;
  step_order: number;
  status: string;
  assignee_id?: number;
  assignee?: UserBasicInfo;
  started_at?: string;
  completed_at?: string;
  comments?: string;
}

export interface SLAInfo {
  sla_name: string;
  response_time: number;
  resolution_time: number;
  due_time: string;
  status: 'active' | 'completed' | 'breached';
}

// ==================== 事件相关类型 ====================

export type IncidentSeverity = 'critical' | 'high' | 'medium' | 'low';
export type IncidentStatus = 'new' | 'investigating' | 'identified' | 'monitoring' | 'resolved' | 'closed';

export interface Incident {
  id: number;
  incident_number: string;
  title: string;
  description: string;
  severity: IncidentSeverity;
  status: IncidentStatus;
  category?: string;
  reporter_id: number;
  assignee_id?: number;
  assignee?: UserBasicInfo;
  impact_analysis?: Record<string, unknown>;
  detected_at: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
}

// ==================== 变更相关类型 ====================

export type ChangeType = 'normal' | 'standard' | 'emergency';
export type ChangeStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'in_progress' | 'completed' | 'rolled_back' | 'cancelled';
export type ChangePriority = 'low' | 'medium' | 'high' | 'critical';
export type ChangeRisk = 'low' | 'medium' | 'high' | 'critical';

export interface Change {
  id: number;
  change_number: string;
  title: string;
  description: string;
  type: ChangeType;
  status: ChangeStatus;
  priority: ChangePriority;
  risk_level: ChangeRisk;
  justification?: string;
  implementation_plan?: string;
  rollback_plan?: string;
  planned_start_date?: string;
  planned_end_date?: string;
  assignee_id?: number;
  assignee?: UserBasicInfo;
  affected_cis?: string[];
  related_tickets?: string[];
  created_at: string;
  updated_at: string;
}

// ==================== 用户相关类型 ====================

export interface UserBasicInfo {
  id: number;
  username: string;
  name: string;
  email: string;
  avatar?: string;
}

export interface User extends UserBasicInfo {
  phone?: string;
  department?: string;
  role?: string;
  status: 'active' | 'inactive' | 'locked';
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

// ==================== 附件相关类型 ====================

export interface Attachment {
  id: number;
  ticket_id?: number;
  file_name: string;
  file_path: string;
  file_url: string;
  file_size: number;
  mime_type: string;
  uploaded_by: number;
  uploader?: UserBasicInfo;
  created_at: string;
}

// ==================== 评论相关类型 ====================

export interface Comment {
  id: number;
  ticket_id: number;
  user_id: number;
  user?: UserBasicInfo;
  content: string;
  is_internal: boolean;
  mentions: number[];
  attachments: number[];
  created_at: string;
  updated_at: string;
}

export interface CommentCreateRequest {
  content: string;
  is_internal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

// ==================== 服务请求相关类型 ====================

export interface ServiceRequest {
  id: number;
  request_number: string;
  catalog_id: number;
  catalog_name?: string;
  title: string;
  reason?: string;
  status: 'pending' | 'approved' | 'rejected' | 'in_progress' | 'completed' | 'cancelled';
  requester_id: number;
  requester?: UserBasicInfo;
  form_data?: Record<string, unknown>;
  approval_status?: string;
  created_at: string;
  updated_at: string;
}

export interface ServiceCatalog {
  id: number;
  name: string;
  description?: string;
  icon?: string;
  parent_id?: number;
  order: number;
  is_active: boolean;
  created_at: string;
}

// ==================== 知识库相关类型 ====================

export interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  summary?: string;
  category_id?: number;
  category_name?: string;
  tags?: string[];
  author_id: number;
  author?: UserBasicInfo;
  view_count: number;
  helpful_count: number;
  status: 'draft' | 'published' | 'archived';
  created_at: string;
  updated_at: string;
}

// ==================== 工作流相关类型 ====================

export interface WorkflowDefinition {
  id: number;
  name: string;
  description?: string;
  xml_content: string;
  version: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WorkflowInstance {
  id: number;
  workflow_id: number;
  workflow?: WorkflowDefinition;
  entity_type: string;
  entity_id: number;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  current_node_id?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

// ==================== 通知相关类型 ====================

export interface Notification {
  id: number;
  user_id: number;
  title: string;
  content: string;
  type: 'info' | 'warning' | 'success' | 'error';
  is_read: boolean;
  read_at?: string;
  data?: Record<string, unknown>;
  created_at: string;
}

// ==================== 审批相关类型 ====================

export interface ApprovalRequest {
  id: number;
  entity_type: string;
  entity_id: number;
  entity_title: string;
  requester_id: number;
  requester?: UserBasicInfo;
  status: 'pending' | 'approved' | 'rejected';
  current_step: number;
  total_steps: number;
  created_at: string;
  updated_at: string;
}

export interface ApprovalActionRequest {
  action: 'approve' | 'reject' | 'delegate';
  comment?: string;
  delegate_to_user_id?: number;
}

// ==================== SLA相关类型 ====================

export interface SLADefinition {
  id: number;
  name: string;
  description?: string;
  response_time: number;
  resolution_time: number;
  priority: TicketPriority;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// ==================== 仪表盘相关类型 ====================

export interface DashboardOverview {
  kpi_metrics: KPIMetric[];
  ticket_trend: TicketTrendData[];
  incident_distribution: IncidentDistributionData[];
  sla_data: SLAComplianceData[];
  satisfaction_data: SatisfactionData[];
  recent_activities: RecentActivity[];
}

export interface KPIMetric {
  id: string;
  title: string;
  value: number;
  unit: string;
  color: string;
  trend: 'up' | 'down' | 'stable';
  change: number;
  change_type: 'increase' | 'decrease' | 'stable';
  description?: string;
}

export interface TicketTrendData {
  date: string;
  open: number;
  in_progress: number;
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
