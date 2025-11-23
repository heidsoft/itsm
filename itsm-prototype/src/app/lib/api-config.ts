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
  page_size?: number;
}

// 分页响应接口
export interface PaginationResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// 租户相关接口
export interface Tenant {
  id: number;
  name: string;
  code: string;
  domain?: string;
  type: 'trial' | 'standard' | 'professional' | 'enterprise';
  status: 'active' | 'suspended' | 'expired' | 'trial';
  created_at: string;
  updated_at: string;
  expires_at?: string;
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
  expires_at?: string;
  settings?: Record<string, unknown>;
}

export interface UpdateTenantRequest {
  name?: string;
  domain?: string;
  type?: string;
  status?: string;
  expires_at?: string;
  settings?: Record<string, unknown>;
}

export interface GetTenantsParams {
  page?: number;
  size?: number;
  status?: string;
  type?: string;
  search?: string;
}

// 工单相关接口（添加租户ID）
export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  type?: string;
  form_fields?: Record<string, unknown>;
  ticket_number: string;
  requester_id: number;
  assignee_id?: number;
  tenant_id: number;
  created_at: string;
  updated_at: string;
  requester?: User;
  assignee?: User;
  tenant?: Tenant;
  // 新增字段
  category?: string;
  subcategory?: string;
  impact?: string;
  urgency?: string;
  resolution?: string;
  work_notes?: string;
  attachments?: Attachment[];
  workflow_steps?: WorkflowStep[];
  comments?: Comment[];
  sla_info?: SLAInfo;
  tags?: string[];
  due_date?: string;
  escalation_level?: number;
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
  content: string;
  type: 'comment' | 'work_note' | 'system';
  created_by: number;
  created_at: string;
  updated_at?: string;
  author?: User;
  is_internal: boolean;
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
  tenant_id?: number;
  role?: string;
  department?: string;
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
  form_fields?: Record<string, unknown>;
  assignee_id?: number;
}

export interface UpdateStatusRequest {
  status: string;
}

export interface GetTicketsParams {
  page?: number;
  size?: number;
  status?: string;
  priority?: string;
  tenant_id?: number;
}

// 服务目录相关接口（添加租户支持）
export interface ServiceCatalog {
  id: number;
  name: string;
  description: string;
  category: string;
  price?: number;
  tenant_id: number;
  is_active: boolean;
  form_schema?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  tenant?: Tenant;
}

export interface ServiceRequest {
  id: number;
  catalog_id: number;
  requester_id: number;
  tenant_id: number;
  status: string;
  reason: string;
  form_data?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
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
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
}

export interface RoleListResponse {
  roles: Role[];
  total: number;
  page: number;
  size: number;
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
  created_at: string;
  updated_at: string;
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
