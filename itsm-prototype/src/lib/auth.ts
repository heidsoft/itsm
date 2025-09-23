/**
 * 认证服务 - 用户认证和权限管理
 * 
 * 功能特性：
 * - 用户登录/登出
 * - JWT Token 管理
 * - 用户信息管理
 * - 权限验证
 * - 自动刷新Token
 */

import { api, TokenManager } from './http-client';

// 用户信息接口
export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  avatar?: string;
  roles: string[];
  permissions: string[];
  tenant_id: number;
  tenant_name: string;
  created_at: string;
  updated_at: string;
}

// 登录请求接口
export interface LoginRequest {
  username: string;
  password: string;
  remember_me?: boolean;
}

// 登录响应接口
export interface LoginResponse {
  user: User;
  token: string;
  refresh_token: string;
  expires_in: number;
}

// 刷新Token响应接口
export interface RefreshTokenResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
}

// 权限检查结果
export interface PermissionCheck {
  hasPermission: boolean;
  reason?: string;
}

// 认证状态
export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
  error: string | null;
}

// 认证服务类
class AuthService {
  private user: User | null = null;
  private refreshTimer: NodeJS.Timeout | null = null;
  private listeners: Array<(state: AuthState) => void> = [];

  constructor() {
    // 初始化时检查本地存储的用户信息
    this.initializeAuth();
  }

  // 初始化认证状态
  private async initializeAuth(): Promise<void> {
    const token = TokenManager.getToken();
    if (token) {
      try {
        // 验证token并获取用户信息
        await this.getCurrentUser();
        this.setupTokenRefresh();
      } catch (error) {
        console.error('Failed to initialize auth:', error);
        this.logout();
      }
    }
  }

  // 添加状态监听器
  addListener(listener: (state: AuthState) => void): () => void {
    this.listeners.push(listener);
    // 返回取消监听的函数
    return () => {
      const index = this.listeners.indexOf(listener);
      if (index > -1) {
        this.listeners.splice(index, 1);
      }
    };
  }

  // 通知状态变化
  private notifyListeners(loading = false, error: string | null = null): void {
    const state: AuthState = {
      isAuthenticated: !!this.user,
      user: this.user,
      loading,
      error,
    };
    
    this.listeners.forEach(listener => {
      try {
        listener(state);
      } catch (error) {
        console.error('Auth listener error:', error);
      }
    });
  }

  // 用户登录
  async login(credentials: LoginRequest): Promise<User> {
    try {
      this.notifyListeners(true);

      const response = await api.post<LoginResponse>('/auth/login', credentials);
      
      // 保存token
      TokenManager.setToken(response.token);
      TokenManager.setRefreshToken(response.refresh_token);
      
      // 保存用户信息
      this.user = response.user;
      
      // 设置自动刷新
      this.setupTokenRefresh(response.expires_in);
      
      this.notifyListeners();
      
      return response.user;
    } catch (error) {
      this.notifyListeners(false, error instanceof Error ? error.message : '登录失败');
      throw error;
    }
  }

  // 用户登出
  async logout(): Promise<void> {
    try {
      // 调用后端登出接口
      if (this.user) {
        await api.post('/auth/logout');
      }
    } catch (error) {
      console.error('Logout API call failed:', error);
    } finally {
      // 清理本地状态
      this.clearAuthState();
    }
  }

  // 清理认证状态
  private clearAuthState(): void {
    TokenManager.clearTokens();
    this.user = null;
    
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
    
    this.notifyListeners();
  }

  // 获取当前用户信息
  async getCurrentUser(): Promise<User> {
    if (!TokenManager.getToken()) {
      throw new Error('No authentication token');
    }

    try {
      const user = await api.get<User>('/auth/me');
      this.user = user;
      this.notifyListeners();
      return user;
    } catch (error) {
      this.clearAuthState();
      throw error;
    }
  }

  // 刷新Token
  async refreshToken(): Promise<void> {
    const refreshToken = TokenManager.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await api.post<RefreshTokenResponse>('/auth/refresh', {
        refresh_token: refreshToken,
      });

      TokenManager.setToken(response.token);
      TokenManager.setRefreshToken(response.refresh_token);
      
      // 重新设置刷新定时器
      this.setupTokenRefresh(response.expires_in);
      
      console.log('Token refreshed successfully');
    } catch (error) {
      console.error('Token refresh failed:', error);
      this.clearAuthState();
      throw error;
    }
  }

  // 设置Token自动刷新
  private setupTokenRefresh(expiresIn?: number): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
    }

    // 默认在token过期前5分钟刷新
    const refreshTime = (expiresIn || 3600) * 1000 - 5 * 60 * 1000;
    
    this.refreshTimer = setTimeout(async () => {
      try {
        await this.refreshToken();
      } catch (error) {
        console.error('Auto refresh token failed:', error);
      }
    }, Math.max(refreshTime, 60000)); // 至少1分钟后刷新
  }

  // 检查是否已认证
  isAuthenticated(): boolean {
    return !!this.user && !!TokenManager.getToken();
  }

  // 获取当前用户
  getUser(): User | null {
    return this.user;
  }

  // 检查用户角色
  hasRole(role: string): boolean {
    return this.user?.roles.includes(role) || false;
  }

  // 检查用户权限
  hasPermission(permission: string): boolean {
    return this.user?.permissions.includes(permission) || false;
  }

  // 检查多个权限（AND关系）
  hasAllPermissions(permissions: string[]): boolean {
    if (!this.user) return false;
    return permissions.every(permission => this.user!.permissions.includes(permission));
  }

  // 检查多个权限（OR关系）
  hasAnyPermission(permissions: string[]): boolean {
    if (!this.user) return false;
    return permissions.some(permission => this.user!.permissions.includes(permission));
  }

  // 详细权限检查
  checkPermission(permission: string): PermissionCheck {
    if (!this.user) {
      return {
        hasPermission: false,
        reason: '用户未登录',
      };
    }

    if (this.user.permissions.includes(permission)) {
      return { hasPermission: true };
    }

    // 检查是否有管理员权限
    if (this.user.roles.includes('admin') || this.user.roles.includes('super_admin')) {
      return { hasPermission: true };
    }

    return {
      hasPermission: false,
      reason: `缺少权限: ${permission}`,
    };
  }

  // 修改密码
  async changePassword(oldPassword: string, newPassword: string): Promise<void> {
    await api.post('/auth/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    });
  }

  // 更新用户信息
  async updateProfile(data: Partial<Pick<User, 'name' | 'email' | 'avatar'>>): Promise<User> {
    const updatedUser = await api.put<User>('/auth/profile', data);
    this.user = updatedUser;
    this.notifyListeners();
    return updatedUser;
  }

  // 获取用户权限列表
  async getUserPermissions(): Promise<string[]> {
    const permissions = await api.get<string[]>('/auth/permissions');
    if (this.user) {
      this.user.permissions = permissions;
      this.notifyListeners();
    }
    return permissions;
  }
}

// 导出单例实例
export const authService = new AuthService();

// React Hook（如果需要在React组件中使用）
export const useAuth = () => {
  // 这个hook需要在React环境中使用
  if (typeof window === 'undefined') {
    return {
      isAuthenticated: false,
      user: null,
      loading: false,
      error: null,
      login: authService.login.bind(authService),
      logout: authService.logout.bind(authService),
      hasRole: authService.hasRole.bind(authService),
      hasPermission: authService.hasPermission.bind(authService),
      hasAllPermissions: authService.hasAllPermissions.bind(authService),
      hasAnyPermission: authService.hasAnyPermission.bind(authService),
      checkPermission: authService.checkPermission.bind(authService),
      changePassword: authService.changePassword.bind(authService),
      updateProfile: authService.updateProfile.bind(authService),
    };
  }

  // 在客户端环境中，这个hook会被React组件使用时重新实现
  return {
    isAuthenticated: authService.isAuthenticated(),
    user: authService.getUser(),
    loading: false,
    error: null,
    login: authService.login.bind(authService),
    logout: authService.logout.bind(authService),
    hasRole: authService.hasRole.bind(authService),
    hasPermission: authService.hasPermission.bind(authService),
    hasAllPermissions: authService.hasAllPermissions.bind(authService),
    hasAnyPermission: authService.hasAnyPermission.bind(authService),
    checkPermission: authService.checkPermission.bind(authService),
    changePassword: authService.changePassword.bind(authService),
    updateProfile: authService.updateProfile.bind(authService),
  };
};

// 权限常量
export const PERMISSIONS = {
  // 事件管理
  INCIDENT_VIEW: 'incident:view',
  INCIDENT_CREATE: 'incident:create',
  INCIDENT_UPDATE: 'incident:update',
  INCIDENT_DELETE: 'incident:delete',
  INCIDENT_ASSIGN: 'incident:assign',
  INCIDENT_CLOSE: 'incident:close',

  // 工单管理
  TICKET_VIEW: 'ticket:view',
  TICKET_CREATE: 'ticket:create',
  TICKET_UPDATE: 'ticket:update',
  TICKET_DELETE: 'ticket:delete',
  TICKET_ASSIGN: 'ticket:assign',
  TICKET_CLOSE: 'ticket:close',

  // 用户管理
  USER_VIEW: 'user:view',
  USER_CREATE: 'user:create',
  USER_UPDATE: 'user:update',
  USER_DELETE: 'user:delete',

  // 系统管理
  SYSTEM_CONFIG: 'system:config',
  SYSTEM_MONITOR: 'system:monitor',
  SYSTEM_BACKUP: 'system:backup',

  // 报表
  REPORT_VIEW: 'report:view',
  REPORT_EXPORT: 'report:export',
} as const;

// 角色常量
export const ROLES = {
  SUPER_ADMIN: 'super_admin',
  ADMIN: 'admin',
  MANAGER: 'manager',
  AGENT: 'agent',
  USER: 'user',
} as const;