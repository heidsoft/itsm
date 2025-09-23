import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// 用户信息接口
export interface User {
  id: number;
  username: string;
  email: string;
  full_name: string;
  role: string;
  status: string;
  avatar?: string;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

// 认证状态接口
interface AuthState {
  // 状态
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  
  // 操作方法
  login: (user: User, token: string) => void;
  logout: () => void;
  updateUser: (user: Partial<User>) => void;
  setLoading: (loading: boolean) => void;
  
  // 权限检查
  hasPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
}

/**
 * 认证状态管理 Store
 * 使用 Zustand 进行全局状态管理，支持持久化存储
 */
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // 初始状态
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,

      // 登录操作
      login: (user: User, token: string) => {
        set({
          user,
          token,
          isAuthenticated: true,
          isLoading: false,
        });
      },

      // 登出操作
      logout: () => {
        set({
          user: null,
          token: null,
          isAuthenticated: false,
          isLoading: false,
        });
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
    }),
    {
      name: 'auth-storage', // localStorage 中的键名
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// 权限常量
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
  
  // 系统权限
  SYSTEM_CONFIG: 'system:config',
  SYSTEM_LOGS: 'system:logs',
  
  // 报告权限
  REPORT_VIEW: 'report:view',
  REPORT_EXPORT: 'report:export',
} as const;

// 角色常量
export const ROLES = {
  ADMIN: 'admin',
  MANAGER: 'manager',
  AGENT: 'agent',
  USER: 'user',
} as const;

// 权限检查 Hook
export const usePermissions = () => {
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const hasRole = useAuthStore((state) => state.hasRole);
  
  return {
    hasPermission,
    hasRole,
    canViewTickets: () => hasPermission(PERMISSIONS.TICKET_VIEW),
    canCreateTickets: () => hasPermission(PERMISSIONS.TICKET_CREATE),
    canUpdateTickets: () => hasPermission(PERMISSIONS.TICKET_UPDATE),
    canDeleteTickets: () => hasPermission(PERMISSIONS.TICKET_DELETE),
    canManageUsers: () => hasPermission(PERMISSIONS.USER_VIEW),
    canViewReports: () => hasPermission(PERMISSIONS.REPORT_VIEW),
    isAdmin: () => hasRole(ROLES.ADMIN),
    isManager: () => hasRole(ROLES.MANAGER),
    isAgent: () => hasRole(ROLES.AGENT),
  };
};