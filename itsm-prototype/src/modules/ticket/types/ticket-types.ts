/**
 * ITSM前端架构 - 模块化类型定义
 * 
 * 工单管理模块类型定义
 * 提供完整的TypeScript类型支持
 */

// ==================== 基础类型 ====================

/**
 * 基础实体接口
 */
export interface BaseEntity {
  id: number;
  created_at: string;
  updated_at: string;
}

/**
 * 分页响应接口
 */
export interface PaginationResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/**
 * API响应接口
 */
export interface ApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
  code?: number;
}

/**
 * 错误响应接口
 */
export interface ErrorResponse {
  success: false;
  error: string;
  code: number;
  details?: Record<string, any>;
}

// ==================== 用户相关类型 ====================

/**
 * 用户接口
 */
export interface User extends BaseEntity {
  username: string;
  email: string;
  full_name: string;
  avatar?: string;
  phone?: string;
  department?: string;
  position?: string;
  role: UserRole;
  status: UserStatus;
  permissions: string[];
  last_login?: string;
  tenant_id?: string;
}

/**
 * 用户角色枚举
 */
export enum UserRole {
  ADMIN = 'admin',
  MANAGER = 'manager',
  TECHNICIAN = 'technician',
  USER = 'user',
  GUEST = 'guest',
}

/**
 * 用户状态枚举
 */
export enum UserStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  SUSPENDED = 'suspended',
  PENDING = 'pending',
}

// ==================== 工单相关类型 ====================

/**
 * 工单状态枚举
 */
export enum TicketStatus {
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
}

/**
 * 工单优先级枚举
 */
export enum TicketPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent',
  CRITICAL = 'critical',
}

/**
 * 工单类型枚举
 */
export enum TicketType {
  INCIDENT = 'incident',
  SERVICE_REQUEST = 'service_request',
  PROBLEM = 'problem',
  CHANGE = 'change',
  QUESTION = 'question',
}

/**
 * 工单接口
 */
export interface Ticket extends BaseEntity {
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  type: TicketType;
  category: string;
  subcategory?: string;
  
  // 关联信息
  assignee_id?: number;
  assignee?: User;
  reporter_id: number;
  reporter: User;
  group_id?: number;
  group?: UserGroup;
  
  // 时间信息
  due_date?: string;
  resolved_at?: string;
  closed_at?: string;
  first_response_at?: string;
  last_response_at?: string;
  
  // SLA信息
  sla_id?: number;
  sla?: SLA;
  sla_status?: SLAStatus;
  sla_violated?: boolean;
  
  // 附加信息
  tags: string[];
  attachments: Attachment[];
  comments: Comment[];
  activities: Activity[];
  
  // 自定义字段
  custom_fields: Record<string, any>;
  
  // 评分
  rating?: number;
  rating_comment?: string;
  
  // 租户信息
  tenant_id?: string;
}

/**
 * 用户组接口
 */
export interface UserGroup extends BaseEntity {
  name: string;
  description?: string;
  members: User[];
  permissions: string[];
  tenant_id?: string;
}

/**
 * SLA接口
 */
export interface SLA extends BaseEntity {
  name: string;
  description?: string;
  priority: TicketPriority;
  response_time: number; // 分钟
  resolution_time: number; // 分钟
  business_hours: BusinessHours;
  escalation_rules: EscalationRule[];
  conditions: SLACondition[];
  active: boolean;
  tenant_id?: string;
}

/**
 * SLA状态枚举
 */
export enum SLAStatus {
  ON_TIME = 'on_time',
  AT_RISK = 'at_risk',
  BREACHED = 'breached',
  NOT_APPLICABLE = 'not_applicable',
}

/**
 * 业务时间接口
 */
export interface BusinessHours {
  timezone: string;
  schedule: {
    [key: string]: {
      start: string;
      end: string;
      enabled: boolean;
    };
  };
  holidays: string[];
}

/**
 * 升级规则接口
 */
export interface EscalationRule {
  id: number;
  name: string;
  trigger_time: number; // 分钟
  actions: EscalationAction[];
  conditions: SLACondition[];
}

/**
 * 升级动作接口
 */
export interface EscalationAction {
  type: 'assign' | 'notify' | 'escalate';
  target: string;
  message?: string;
}

/**
 * SLA条件接口
 */
export interface SLACondition {
  field: string;
  operator: 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'in' | 'not_in';
  value: any;
}

// ==================== 附件相关类型 ====================

/**
 * 附件接口
 */
export interface Attachment extends BaseEntity {
  name: string;
  original_name: string;
  url: string;
  size: number;
  type: string;
  mime_type: string;
  uploaded_by: number;
  uploaded_by_user?: User;
  ticket_id: number;
  is_public: boolean;
}

// ==================== 评论相关类型 ====================

/**
 * 评论接口
 */
export interface Comment extends BaseEntity {
  content: string;
  author_id: number;
  author: User;
  ticket_id: number;
  is_internal: boolean;
  is_system: boolean;
  attachments: Attachment[];
  mentions: User[];
}

// ==================== 活动相关类型 ====================

/**
 * 活动类型枚举
 */
export enum ActivityType {
  CREATED = 'created',
  UPDATED = 'updated',
  ASSIGNED = 'assigned',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  REOPENED = 'reopened',
  COMMENTED = 'commented',
  ATTACHMENT_ADDED = 'attachment_added',
  STATUS_CHANGED = 'status_changed',
  PRIORITY_CHANGED = 'priority_changed',
}

/**
 * 活动接口
 */
export interface Activity extends BaseEntity {
  type: ActivityType;
  description: string;
  user_id?: number;
  user?: User;
  ticket_id: number;
  old_value?: any;
  new_value?: any;
  metadata?: Record<string, any>;
}

// ==================== 过滤和搜索类型 ====================

/**
 * 工单过滤器接口
 */
export interface TicketFilters {
  status?: TicketStatus[];
  priority?: TicketPriority[];
  type?: TicketType[];
  category?: string[];
  assignee_id?: number[];
  reporter_id?: number[];
  group_id?: number[];
  tags?: string[];
  sla_status?: SLAStatus[];
  date_range?: {
    field: 'created_at' | 'updated_at' | 'due_date' | 'resolved_at' | 'closed_at';
    start: string;
    end: string;
  };
  search?: string;
  custom_fields?: Record<string, any>;
}

/**
 * 排序配置接口
 */
export interface SortConfig {
  field: string;
  order: 'asc' | 'desc';
}

/**
 * 查询参数接口
 */
export interface QueryParams {
  page?: number;
  pageSize?: number;
  filters?: TicketFilters;
  sort?: SortConfig[];
  search?: string;
}

// ==================== 表单相关类型 ====================

/**
 * 创建工单请求接口
 */
export interface CreateTicketRequest {
  title: string;
  description: string;
  type: TicketType;
  priority: TicketPriority;
  category: string;
  subcategory?: string;
  assignee_id?: number;
  group_id?: number;
  due_date?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  attachments?: File[];
}

/**
 * 更新工单请求接口
 */
export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  subcategory?: string;
  assignee_id?: number;
  group_id?: number;
  due_date?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
}

/**
 * 分配工单请求接口
 */
export interface AssignTicketRequest {
  assignee_id: number;
  comment?: string;
  notify?: boolean;
}

/**
 * 解决工单请求接口
 */
export interface ResolveTicketRequest {
  resolution: string;
  category?: string;
  comment?: string;
  notify?: boolean;
}

/**
 * 关闭工单请求接口
 */
export interface CloseTicketRequest {
  reason?: string;
  rating?: number;
  comment?: string;
  notify?: boolean;
}

/**
 * 重新打开工单请求接口
 */
export interface ReopenTicketRequest {
  reason?: string;
  comment?: string;
  notify?: boolean;
}

// ==================== 统计相关类型 ====================

/**
 * 工单统计接口
 */
export interface TicketStats {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
  overdue: number;
  by_priority: Record<TicketPriority, number>;
  by_status: Record<TicketStatus, number>;
  by_category: Record<string, number>;
  by_assignee: Record<number, number>;
  avg_resolution_time: number;
  avg_first_response_time: number;
  sla_compliance_rate: number;
}

/**
 * 时间范围统计接口
 */
export interface TimeRangeStats {
  period: 'day' | 'week' | 'month' | 'quarter' | 'year';
  start_date: string;
  end_date: string;
  created: number;
  resolved: number;
  closed: number;
  avg_resolution_time: number;
  trends: {
    date: string;
    created: number;
    resolved: number;
    closed: number;
  }[];
}

// ==================== 通知相关类型 ====================

/**
 * 通知类型枚举
 */
export enum NotificationType {
  EMAIL = 'email',
  SMS = 'sms',
  PUSH = 'push',
  IN_APP = 'in_app',
  WEBHOOK = 'webhook',
}

/**
 * 通知模板接口
 */
export interface NotificationTemplate {
  id: number;
  name: string;
  type: NotificationType;
  subject?: string;
  content: string;
  variables: string[];
  active: boolean;
  tenant_id?: string;
}

/**
 * 通知接口
 */
export interface Notification extends BaseEntity {
  type: NotificationType;
  recipient_id: number;
  recipient?: User;
  subject?: string;
  content: string;
  sent_at?: string;
  read_at?: string;
  failed_at?: string;
  error_message?: string;
  metadata?: Record<string, any>;
}

// ==================== 导出类型 ====================

export type {
  BaseEntity,
  PaginationResponse,
  ApiResponse,
  ErrorResponse,
  User,
  UserRole,
  UserStatus,
  TicketStatus,
  TicketPriority,
  TicketType,
  Ticket,
  UserGroup,
  SLA,
  SLAStatus,
  BusinessHours,
  EscalationRule,
  EscalationAction,
  SLACondition,
  Attachment,
  Comment,
  ActivityType,
  Activity,
  TicketFilters,
  SortConfig,
  QueryParams,
  CreateTicketRequest,
  UpdateTicketRequest,
  AssignTicketRequest,
  ResolveTicketRequest,
  CloseTicketRequest,
  ReopenTicketRequest,
  TicketStats,
  TimeRangeStats,
  NotificationType,
  NotificationTemplate,
  Notification,
};

// ==================== 类型守卫 ====================

/**
 * 类型守卫函数
 */
export const TypeGuards = {
  isTicket: (obj: any): obj is Ticket => {
    return obj && typeof obj.id === 'number' && typeof obj.title === 'string';
  },
  
  isUser: (obj: any): obj is User => {
    return obj && typeof obj.id === 'number' && typeof obj.username === 'string';
  },
  
  isComment: (obj: any): obj is Comment => {
    return obj && typeof obj.id === 'number' && typeof obj.content === 'string';
  },
  
  isAttachment: (obj: any): obj is Attachment => {
    return obj && typeof obj.id === 'number' && typeof obj.name === 'string';
  },
  
  isActivity: (obj: any): obj is Activity => {
    return obj && typeof obj.id === 'number' && typeof obj.type === 'string';
  },
};

export default TypeGuards;
