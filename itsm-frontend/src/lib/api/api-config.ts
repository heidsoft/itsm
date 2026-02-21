// API 基础配置
// 在客户端使用相对路径以利用 Next.js 的 Rewrites 代理
// 在服务端（SSR）使用完整的后端地址
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || (typeof window === 'undefined' ? 'http://localhost:8090' : '');
export const API_VERSION = process.env.NEXT_PUBLIC_API_VERSION || 'v1';
export const API_TIMEOUT = parseInt(process.env.NEXT_PUBLIC_API_TIMEOUT || '30000');

// 通用 API 响应接口
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// API 错误码定义
export const API_ERROR_CODES = {
  SUCCESS: 0,
  PARAM_ERROR: 1001,
  VALIDATION_ERROR: 1002,
  AUTH_FAILED: 2001,
  FORBIDDEN: 2003,
  NOT_FOUND: 4004,
  INTERNAL_ERROR: 5001,
} as const;

// 分页请求接口
export interface PaginationRequest {
  page?: number;
  pageSize?: number;
}

// 分页响应接口
export interface PaginationResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 租户相关接口
export interface Tenant {
  id: number;
  name: string;
  code: string;
  domain?: string;
  type: 'trial' | 'standard' | 'professional' | 'enterprise';
  status: 'active' | 'suspended' | 'expired' | 'trial';
  createdAt: string;
  updatedAt: string;
  expiresAt?: string;
  settings?: Record<string, unknown>;
}

export interface TenantListResponse {
  tenants: Tenant[];
  total: number;
  page: number;
  size: number;
}

export interface CreateTenantRequest {
  name: string;
  code: string;
  domain?: string;
  type: string;
  expiresAt?: string;
  settings?: Record<string, unknown>;
}

export interface UpdateTenantRequest {
  name?: string;
  domain?: string;
  type?: string;
  status?: string;
  expiresAt?: string;
  settings?: Record<string, unknown>;
}

export interface GetTenantsParams {
  page?: number;
  size?: number;
  status?: string;
  type?: string;
  search?: string;
}

// 重新导出标准Ticket类型，并扩展租户相关字段
import type { Ticket as BaseTicket } from './types';

export interface Ticket extends BaseTicket {
  tenant_id?: number;
  tenant?: Tenant;
  // 扩展字段
  subcategory?: string;
  impact?: string;
  urgency?: string;
  work_notes?: string;
  dueDate?: string; // 兼容旧字段名
  escalation_level?: number;
  source?: string;
  business_value?: string;
  custom_fields?: Record<string, unknown>;
}

// 附件接口
export interface Attachment {
  id: number;
  filename: string;
  original_name: string;
  file_size: number;
  mime_type: string;
  url: string;
  uploaded_by: number;
  uploaded_at: string;
  uploader?: User;
}

// 工作流步骤接口
export interface WorkflowStep {
  id: number;
  step_name: string;
  step_order: number;
  status: 'pending' | 'in_progress' | 'completed' | 'skipped';
  assignee_id?: number;
  assignee?: User;
  started_at?: string;
  completed_at?: string;
  comments?: string;
  required_approval: boolean;
  approval_status?: 'pending' | 'approved' | 'rejected';
  approval_comments?: string;
}

// 评论接口
export interface Comment {
  id: number;
  ticket_id?: number;
  content: string;
  type?: 'comment' | 'work_note' | 'system';
  created_by?: number;
  user_id?: number;
  created_at: string;
  updated_at?: string;
  author?: User;
  is_internal: boolean;
  mentions?: number[];
  attachments?: number[];
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenant_id?: number;
  };
}

// SLA信息接口
export interface SLAInfo {
  sla_id: number;
  sla_name: string;
  response_time: number; // 分钟
  resolution_time: number; // 分钟
  start_time: string;
  due_time: string;
  breach_time?: string;
  status: 'active' | 'breached' | 'completed';
}

export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  tenantId?: number;
  role?: string;
  department?: string;
  permissions?: string[];
  createdAt?: string;
  updatedAt?: string;
}

export interface TicketListResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  size: number;
}

export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  category?: string;
  category_id?: number;
  formFields?: Record<string, unknown>;
  assigneeId?: number;
}

export interface UpdateStatusRequest {
  status: string;
}

export interface GetTicketsParams {
  page?: number;
  size?: number;
  status?: string;
  priority?: string;
  tenantId?: number;
}

// 服务目录相关接口（添加租户支持）
export interface ServiceCatalog {
  id: number;
  name: string;
  description: string;
  category: string;
  price?: number;
  tenantId: number;
  isActive: boolean;
  formSchema?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  tenant?: Tenant;
}

export interface ServiceRequest {
  id: number;
  catalogId: number;
  requesterId: number;
  tenantId: number;
  status: string;
  reason: string;
  formData?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  catalog?: ServiceCatalog;
  requester?: User;
  tenant?: Tenant;
}

// 角色相关接口
export interface Role {
  id: number;
  name: string;
  description: string;
  permissions: string[];
  status?: 'active' | 'inactive';
  isSystem?: boolean;
  is_system?: boolean; // Compatible with backend snake_case
  userCount?: number;
  createdAt: string;
  updatedAt: string;
}

export interface RoleListResponse {
  roles: Role[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateRoleRequest {
  name: string;
  description: string;
  permissions: string[];
  status: 'active' | 'inactive';
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  permissions?: string[];
  status?: 'active' | 'inactive';
}

export interface GetRolesParams {
  page?: number;
  size?: number;
  status?: string;
  search?: string;
}

// 系统配置相关接口
export interface SystemConfig {
  id: number;
  key: string;
  value: string;
  description?: string;
  category: string;
  createdAt: string;
  updatedAt: string;
}

export interface SystemConfigListResponse {
  configs: SystemConfig[];
  total: number;
}

export interface UpdateSystemConfigRequest {
  key: string;
  value: string;
  description?: string;
}

export interface GetSystemConfigsParams {
  category?: string;
  search?: string;
}
