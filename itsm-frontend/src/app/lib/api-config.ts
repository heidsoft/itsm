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

// 工单相关接口（添加租户ID）
export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  type?: string;
  formFields?: Record<string, unknown>;
  ticketNumber: string;
  requesterId: number;
  assigneeId?: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
  requester?: User;
  assignee?: User;
  tenant?: Tenant;
  // 新增字段
  category?: string;
  subcategory?: string;
  impact?: string;
  urgency?: string;
  resolution?: string;
  workNotes?: string;
  attachments?: Attachment[];
  workflowSteps?: WorkflowStep[];
  comments?: Comment[];
  slaInfo?: SLAInfo;
  tags?: string[];
  dueDate?: string;
  escalationLevel?: number;
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
  content: string;
  type: 'comment' | 'work_note' | 'system';
  createdBy: number;
  createdAt: string;
  updatedAt?: string;
  author?: User;
  isInternal: boolean;
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
