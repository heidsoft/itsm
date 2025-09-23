/**
 * 用户相关类型定义
 */

export type UserRole = 'admin' | 'manager' | 'agent' | 'technician' | 'end_user';
export type UserStatus = 'active' | 'inactive' | 'suspended' | 'pending';

export interface User {
  id: number;
  username: string;
  email: string;
  fullName: string;
  firstName?: string;
  lastName?: string;
  avatar?: string;
  phone?: string;
  role: UserRole;
  status: UserStatus;
  department?: string;
  jobTitle?: string;
  manager?: User;
  location?: string;
  timezone?: string;
  language?: string;
  
  // 权限相关
  permissions: string[];
  groups: UserGroup[];
  
  // 时间戳
  createdAt: string;
  updatedAt: string;
  lastLoginAt?: string;
  
  // 偏好设置
  preferences: UserPreferences;
  
  // 元数据
  tenantId: number;
  isActive: boolean;
  emailVerified: boolean;
  phoneVerified: boolean;
}

export interface UserGroup {
  id: number;
  name: string;
  description?: string;
  permissions: string[];
  isActive: boolean;
}

export interface UserPreferences {
  theme: 'light' | 'dark' | 'auto';
  language: string;
  timezone: string;
  dateFormat: string;
  timeFormat: '12h' | '24h';
  
  // 通知偏好
  notifications: {
    email: boolean;
    sms: boolean;
    inApp: boolean;
    desktop: boolean;
    
    // 通知类型
    ticketAssigned: boolean;
    ticketUpdated: boolean;
    ticketEscalated: boolean;
    ticketResolved: boolean;
    slaBreached: boolean;
    systemMaintenance: boolean;
  };
  
  // 界面偏好
  ui: {
    sidebarCollapsed: boolean;
    tablePageSize: number;
    defaultView: 'table' | 'card' | 'kanban';
    showAvatars: boolean;
    compactMode: boolean;
  };
  
  // 工作偏好
  work: {
    autoAssign: boolean;
    defaultPriority: string;
    workingHours: {
      start: string;
      end: string;
      timezone: string;
    };
    workingDays: number[]; // 0-6, 0 = Sunday
  };
}

export interface CreateUserRequest {
  username: string;
  email: string;
  fullName: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  role: UserRole;
  department?: string;
  jobTitle?: string;
  managerId?: number;
  location?: string;
  timezone?: string;
  language?: string;
  groupIds?: number[];
  sendWelcomeEmail?: boolean;
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
  fullName?: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  role?: UserRole;
  status?: UserStatus;
  department?: string;
  jobTitle?: string;
  managerId?: number;
  location?: string;
  timezone?: string;
  language?: string;
  groupIds?: number[];
  avatar?: string;
}

export interface UserFilters {
  role?: UserRole[];
  status?: UserStatus[];
  department?: string[];
  location?: string[];
  groupId?: number[];
  search?: string;
  isActive?: boolean;
  lastLoginAfter?: string;
  lastLoginBefore?: string;
  createdAfter?: string;
  createdBefore?: string;
}

export interface UserListResponse {
  users: User[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface UserStats {
  total: number;
  active: number;
  inactive: number;
  suspended: number;
  byRole: Record<UserRole, number>;
  byDepartment: Record<string, number>;
  newThisMonth: number;
  activeThisWeek: number;
}

// 认证相关
export interface LoginRequest {
  username: string;
  password: string;
  rememberMe?: boolean;
  captcha?: string;
}

export interface LoginResponse {
  user: User;
  token: string;
  refreshToken: string;
  expiresIn: number;
  permissions: string[];
}

export interface RefreshTokenRequest {
  refreshToken: string;
}

export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

export interface ResetPasswordRequest {
  email: string;
  captcha?: string;
}

export interface SetPasswordRequest {
  token: string;
  password: string;
  confirmPassword: string;
}

// 用户活动
export interface UserActivity {
  id: number;
  userId: number;
  type: 'login' | 'logout' | 'password_change' | 'profile_update' | 'permission_change';
  description: string;
  ipAddress: string;
  userAgent: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
}

// 用户会话
export interface UserSession {
  id: string;
  userId: number;
  ipAddress: string;
  userAgent: string;
  location?: string;
  isActive: boolean;
  createdAt: string;
  lastActivityAt: string;
  expiresAt: string;
}

// 权限相关
export interface Permission {
  id: number;
  name: string;
  description: string;
  resource: string;
  action: string;
  isActive: boolean;
}

export interface RolePermission {
  role: UserRole;
  permissions: Permission[];
}

// 用户导入/导出
export interface UserImportRequest {
  file: File;
  options: {
    skipHeader: boolean;
    updateExisting: boolean;
    sendWelcomeEmail: boolean;
    defaultRole: UserRole;
    defaultStatus: UserStatus;
  };
}

export interface UserImportResult {
  total: number;
  success: number;
  failed: number;
  errors: Array<{
    row: number;
    field: string;
    error: string;
  }>;
  createdUsers: User[];
  updatedUsers: User[];
}

export interface UserExportRequest {
  filters?: UserFilters;
  fields?: string[];
  format: 'csv' | 'xlsx' | 'json';
}