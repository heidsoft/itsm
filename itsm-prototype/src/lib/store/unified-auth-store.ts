/**
 * 统一的认证状态管理 Store
 * 合并了 tenant 支持和 permissions 系统
 * 使用 Zustand 进行全局状态管理，支持持久化存储
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { httpClient } from '@/app/lib/http-client';

// ===================================
// 类型定义
// ===================================

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

export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  role: string;
  status?: string;
  avatar?: string;
  tenant_id?: number;
  department?: string;
  phone?: string;
  permissions?: string[];
  created_at?: string;
  updated_at?: string;
}

interface AuthState {
  // 状态
  user: User | null;
  token: string | null;
  currentTenant: Tenant | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  
  // 认证操作
  login: (user: User, token: string, tenant?: Tenant) => void;
  logout: () => void;
  updateUser: (user: Partial<User>) => void;
  setLoading: (loading: boolean) => void;
  
  // 租户操作
  setCurrentTenant: (tenant: Tenant) => void;
  clearTenant: () => void;
  
  // 权限检查
  hasPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
  isAdmin: () => boolean;
}

// ===================================
// Store 定义
// ===================================

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // 初始状态
      user: null,
      token: null,
      currentTenant: null,
      isAuthenticated: false,
      isLoading: false,

      // 登录操作
      login: (user: User, token: string, tenant?: Tenant) => {
        set({
          user,
          token,
          isAuthenticated: true,
          isLoading: false,
          currentTenant: tenant || null,
        });
        
        // 同步到 localStorage 和 httpClient
        if (typeof window !== 'undefined') {
          localStorage.setItem('access_token', token);
        }
        httpClient.setToken(token);
        
        if (tenant) {
          httpClient.setTenantId(tenant.id);
          httpClient.setTenantCode(tenant.code);
        }
      },

      // 登出操作
      logout: () => {
        set({
          user: null,
          token: null,
          isAuthenticated: false,
          isLoading: false,
          currentTenant: null,
        });
        
        // 清除所有认证信息
        if (typeof window !== 'undefined') {
          localStorage.removeItem('access_token');
          localStorage.removeItem('auth_token');
          localStorage.removeItem('refresh_token');
          localStorage.removeItem('current_tenant_id');
          localStorage.removeItem('current_tenant_code');
        }
        
        httpClient.clearToken();
        httpClient.setTenantId(null);
        httpClient.setTenantCode(null);
      },

      // 更新用户信息
      updateUser: (userData: Partial<User>) => {
        const { user } = get();
        if (user) {
          set({
            user: { ...user, ...userData },
          });
        }
      },

      // 设置加载状态
      setLoading: (loading: boolean) => {
        set({ isLoading: loading });
      },

      // 设置当前租户
      setCurrentTenant: (tenant: Tenant) => {
        set({ currentTenant: tenant });
        httpClient.setTenantId(tenant.id);
        httpClient.setTenantCode(tenant.code);
        
        if (typeof window !== 'undefined') {
          localStorage.setItem('current_tenant_id', tenant.id.toString());
          localStorage.setItem('current_tenant_code', tenant.code);
        }
      },

      // 清除租户
      clearTenant: () => {
        set({ currentTenant: null });
        httpClient.setTenantId(null);
        httpClient.setTenantCode(null);
        
        if (typeof window !== 'undefined') {
          localStorage.removeItem('current_tenant_id');
          localStorage.removeItem('current_tenant_code');
        }
      },

      // 检查用户权限
      hasPermission: (permission: string) => {
        const { user } = get();
        return user?.permissions?.includes(permission) || false;
      },

      // 检查用户角色
      hasRole: (role: string) => {
        const { user } = get();
        return user?.role === role;
      },

      // 检查是否为管理员
      isAdmin: () => {
        const { user } = get();
        return user?.role === 'admin' || user?.role === 'super_admin';
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        currentTenant: state.currentTenant,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// ===================================
// 租户管理 Store
// ===================================

interface TenantState {
  tenants: Tenant[];
  loading: boolean;
  error: string | null;
  setTenants: (tenants: Tenant[]) => void;
  addTenant: (tenant: Tenant) => void;
  updateTenant: (id: number, tenant: Partial<Tenant>) => void;
  removeTenant: (id: number) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useTenantStore = create<TenantState>((set) => ({
  tenants: [],
  loading: false,
  error: null,
  setTenants: (tenants) => set({ tenants }),
  addTenant: (tenant) => set((state) => ({ tenants: [...state.tenants, tenant] })),
  updateTenant: (id, updatedTenant) =>
    set((state) => ({
      tenants: state.tenants.map((tenant) =>
        tenant.id === id ? { ...tenant, ...updatedTenant } : tenant
      ),
    })),
  removeTenant: (id) =>
    set((state) => ({
      tenants: state.tenants.filter((tenant) => tenant.id !== id),
    })),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
}));

// ===================================
// 权限常量
// ===================================

export const PERMISSIONS = {
  // 工单权限
  TICKET_VIEW: 'ticket:view',
  TICKET_CREATE: 'ticket:create',
  TICKET_UPDATE: 'ticket:update',
  TICKET_DELETE: 'ticket:delete',
  TICKET_ASSIGN: 'ticket:assign',
  TICKET_CLOSE: 'ticket:close',
  
  // 用户权限
  USER_VIEW: 'user:view',
  USER_CREATE: 'user:create',
  USER_UPDATE: 'user:update',
  USER_DELETE: 'user:delete',
  
  // 事件权限
  INCIDENT_VIEW: 'incident:view',
  INCIDENT_CREATE: 'incident:create',
  INCIDENT_UPDATE: 'incident:update',
  INCIDENT_DELETE: 'incident:delete',
  
  // 系统权限
  SYSTEM_CONFIG: 'system:config',
  SYSTEM_LOGS: 'system:logs',
  
  // 报告权限
  REPORT_VIEW: 'report:view',
  REPORT_EXPORT: 'report:export',
  
  // 审计权限
  AUDIT_VIEW: 'audit:view',
  AUDIT_EXPORT: 'audit:export',
} as const;

// 角色常量
export const ROLES = {
  SUPER_ADMIN: 'super_admin',
  ADMIN: 'admin',
  MANAGER: 'manager',
  AGENT: 'agent',
  TECHNICIAN: 'technician',
  END_USER: 'end_user',
  USER: 'user', // 兼容旧版本
} as const;

// ===================================
// 权限检查 Hook
// ===================================

export const usePermissions = () => {
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const hasRole = useAuthStore((state) => state.hasRole);
  const isAdmin = useAuthStore((state) => state.isAdmin);
  
  return {
    // 基础权限检查
    hasPermission,
    hasRole,
    isAdmin,
    
    // 工单权限
    canViewTickets: () => hasPermission(PERMISSIONS.TICKET_VIEW) || isAdmin(),
    canCreateTickets: () => hasPermission(PERMISSIONS.TICKET_CREATE) || isAdmin(),
    canUpdateTickets: () => hasPermission(PERMISSIONS.TICKET_UPDATE) || isAdmin(),
    canDeleteTickets: () => hasPermission(PERMISSIONS.TICKET_DELETE) || isAdmin(),
    canAssignTickets: () => hasPermission(PERMISSIONS.TICKET_ASSIGN) || isAdmin(),
    
    // 用户权限
    canViewUsers: () => hasPermission(PERMISSIONS.USER_VIEW) || isAdmin(),
    canManageUsers: () => hasPermission(PERMISSIONS.USER_CREATE) || isAdmin(),
    
    // 事件权限
    canViewIncidents: () => hasPermission(PERMISSIONS.INCIDENT_VIEW) || isAdmin(),
    canManageIncidents: () => hasPermission(PERMISSIONS.INCIDENT_CREATE) || isAdmin(),
    
    // 报告权限
    canViewReports: () => hasPermission(PERMISSIONS.REPORT_VIEW) || isAdmin(),
    canExportReports: () => hasPermission(PERMISSIONS.REPORT_EXPORT) || isAdmin(),
    
    // 角色检查
    isSuperAdmin: () => hasRole(ROLES.SUPER_ADMIN),
    isManager: () => hasRole(ROLES.MANAGER),
    isAgent: () => hasRole(ROLES.AGENT),
    isTechnician: () => hasRole(ROLES.TECHNICIAN),
    isEndUser: () => hasRole(ROLES.END_USER) || hasRole(ROLES.USER),
  };
};

// ===================================
// 导出兼容性别名
// ===================================

// 为了向后兼容，导出类型
export type { AuthState, TenantState };

