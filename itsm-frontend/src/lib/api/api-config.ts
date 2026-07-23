// Browser requests default to same-origin so production traffic goes through the
// reverse proxy. Server-side requests are redirected to ITSM_BACKEND_URL by
// HttpClient without exposing the container hostname to the browser bundle.
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';
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
  type:
    | 'standard'
    | 'internal'
    | 'saas_customer'
    | 'msp_provider'
    | 'msp_customer'
    | 'msp'
    | 'customer'
    // Legacy frontend/session values kept for compatibility with existing auth state.
    | 'trial'
    | 'professional'
    | 'enterprise';
  status: 'active' | 'suspended' | 'expired' | 'deleted';
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
  tenantId?: number;
  templateId?: number;
  tenant?: Tenant;
  // 扩展字段
  subcategory?: string;
  impact?: string;
  urgency?: string;
  workNotes?: string;
  dueDate?: string; // 兼容旧字段名
  escalationLevel?: number;
  source?: string;
  businessValue?: string;
  customFields?: Record<string, unknown>;
}

// 附件接口
export interface Attachment {
  id: number;
  filename: string;
  originalName: string;
  fileSize: number;
  mimeType: string;
  url: string;
  uploadedBy: number;
  uploadedAt: string;
  uploader?: User;
}

// 工作流步骤接口
export interface WorkflowStep {
  id: number;
  stepName: string;
  stepOrder: number;
  status: 'pending' | 'in_progress' | 'completed' | 'skipped';
  assigneeId?: number;
  assignee?: User;
  startedAt?: string;
  completedAt?: string;
  comments?: string;
  requiredApproval: boolean;
  approvalStatus?: 'pending' | 'approved' | 'rejected';
  approvalComments?: string;
}

// 评论接口
export interface Comment {
  id: number;
  ticketId?: number;
  content: string;
  type?: 'comment' | 'work_note' | 'system';
  createdBy?: number;
  userId?: number;
  createdAt: string;
  updatedAt?: string;
  author?: User;
  isInternal: boolean;
  mentions?: number[];
  attachments?: number[];
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenantId?: number;
  };
}

// SLA信息接口
export interface SLAInfo {
  slaId: number;
  slaName: string;
  responseTime: number; // 分钟
  resolutionTime: number; // 分钟
  startTime: string;
  dueTime: string;
  breachTime?: string;
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
  pageSize?: number;
  size: number;
  totalPages?: number;
}

export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  type?: 'incident' | 'service_request' | 'change' | 'problem' | string;
  category?: string;
  categoryId?: number;
  formFields?: Record<string, unknown>;
  assigneeId?: number;
  workflowDefinitionKey?: string;
}

export interface UpdateStatusRequest {
  status: string;
}

export interface GetTicketsParams {
  page?: number;
  pageSize?: number;
  size?: number;
  status?: string;
  priority?: string;
  tenantId?: number;
  templateId?: number;
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
  code?: string;
  description: string;
  permissions: string[];
  status?: 'active' | 'inactive';
  isSystem?: boolean;
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
  code?: string;
  description: string;
  permissions: string[];
  status: 'active' | 'inactive';
}

export interface UpdateRoleRequest {
  name?: string;
  code?: string;
  description?: string;
  permissions?: string[];
  status?: 'active' | 'inactive';
}

export interface GetRolesParams {
  page?: number;
  size?: number;
  pageSize?: number;
  status?: string;
  search?: string;
}

export interface PermissionCatalogItem {
  id: number;
  code: string;
  name: string;
  description?: string;
  resource: string;
  action: string;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
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
  configs?: SystemConfig[];
  items?: SystemConfig[];
  total: number;
  page?: number;
  size?: number;
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
