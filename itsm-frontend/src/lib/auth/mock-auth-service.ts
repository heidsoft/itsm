/**
 * 模拟认证服务 - 用于前端开发和测试
 */

import { useAuthStore, User } from '../store/auth-store';
import {
  STORAGE_KEYS,
  clearAuthStorage,
  migrateLegacyAuthStorage,
  setAccessToken,
  setRefreshToken,
} from './token-storage';

interface LoginRequest {
  username: string;
  password: string;
}

interface LoginResponse {
  user: User;
  token: string;
  refresh_token: string;
  expires_in: number;
}

interface RefreshTokenResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
}

interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

// 模拟用户数据
const MOCK_USERS = [
  {
    id: 1,
    username: 'admin',
    password: 'admin123',
    email: 'admin@itsm.com',
    full_name: '系统管理员',
    name: '系统管理员',
    role: 'admin',
    status: 'active',
    avatar: '',
    permissions: ['ticket:view', 'ticket:create', 'ticket:update', 'ticket:delete', 'user:view', 'user:create', 'system:config'],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  },
  {
    id: 2,
    username: 'user',
    password: 'user123',
    email: 'user@itsm.com',
    full_name: '普通用户',
    name: '普通用户',
    role: 'user',
    status: 'active',
    avatar: '',
    permissions: ['ticket:view', 'ticket:create'],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }
];

export class MockAuthService {
  private static readonly TOKEN_KEY = STORAGE_KEYS.ACCESS_TOKEN;
  private static readonly REFRESH_TOKEN_KEY = STORAGE_KEYS.REFRESH_TOKEN;
  private static refreshTimer: NodeJS.Timeout | null = null;

  /**
   * 模拟用户登录
   */
  static async login(credentials: LoginRequest): Promise<LoginResponse> {
    // 模拟网络延迟
    await new Promise(resolve => setTimeout(resolve, 1000));

    // 查找用户
    const user = MOCK_USERS.find(u => 
      u.username === credentials.username && u.password === credentials.password
    );

    if (!user) {
      throw new Error('用户名或密码错误');
    }

    if (user.status !== 'active') {
      throw new Error('账户已被禁用');
    }

    // 生成模拟token
    const token = this.generateMockToken(user);
    const refreshToken = this.generateMockRefreshToken(user);
    const expiresIn = 3600; // 1小时

    // 保存认证信息
    this.setTokens(token, refreshToken);

    // 构造用户对象（移除密码）
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { password, ...userWithoutPassword } = user;
    const userResponse: User = userWithoutPassword;

    // 更新认证状态
    useAuthStore.getState().login(userResponse, token);

    // 设置自动刷新token
    this.setupTokenRefresh(expiresIn);

    return {
      user: userResponse,
      token,
      refresh_token: refreshToken,
      expires_in: expiresIn
    };
  }

  /**
   * 用户登出
   */
  static async logout(): Promise<void> {
    // 模拟网络延迟
    await new Promise(resolve => setTimeout(resolve, 500));

    // 清除本地认证信息
    this.clearTokens();
    
    // 清除认证状态
    useAuthStore.getState().logout();
    
    // 清除自动刷新定时器
    this.clearTokenRefresh();
  }

  /**
   * 刷新访问token
   */
  static async refreshToken(): Promise<RefreshTokenResponse> {
    const refreshToken = this.getRefreshToken();
    
    if (!refreshToken) {
      throw new Error('Refresh token not found');
    }

    // 模拟网络延迟
    await new Promise(resolve => setTimeout(resolve, 500));

    // 验证refresh token（简单模拟）
    if (!this.isValidRefreshToken(refreshToken)) {
      throw new Error('Invalid refresh token');
    }

    // 生成新的token
    const newToken = this.generateMockToken();
    const newRefreshToken = this.generateMockRefreshToken();
    const expiresIn = 3600;

    // 保存新的token
    this.setTokens(newToken, newRefreshToken);

    return {
      token: newToken,
      refresh_token: newRefreshToken,
      expires_in: expiresIn
    };
  }

  /**
   * 获取当前用户信息
   */
  static async getCurrentUser(): Promise<User> {
    const token = this.getToken();
    
    if (!token) {
      throw new Error('No authentication token');
    }

    // 模拟网络延迟
    await new Promise(resolve => setTimeout(resolve, 300));

    // 从token中解析用户信息（简单模拟）
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      const user = MOCK_USERS.find(u => u.id === payload.user_id);
      
      if (!user) {
        throw new Error('User not found');
      }

      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { password, ...userWithoutPassword } = user;
      return userWithoutPassword;
    } catch {
      throw new Error('Invalid token');
    }
  }

  /**
   * 修改密码
   */
  static async changePassword(data: ChangePasswordRequest): Promise<void> {
    // 模拟网络延迟
    await new Promise(resolve => setTimeout(resolve, 1000));

    if (data.new_password !== data.confirm_password) {
      throw new Error('新密码和确认密码不匹配');
    }

    // 这里只是模拟，实际应该验证当前密码并更新
    console.log('Password changed successfully (mock)');
  }

  /**
   * 检查是否已认证
   */
  static isAuthenticated(): boolean {
    const token = this.getToken();
    return token !== null && !this.isTokenExpired(token);
  }

  /**
   * 获取访问token
   */
  static getToken(): string | null {
    if (typeof window === 'undefined') return null;
    migrateLegacyAuthStorage();
    return localStorage.getItem(this.TOKEN_KEY);
  }

  /**
   * 获取刷新token
   */
  static getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    migrateLegacyAuthStorage();
    return localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  // 私有方法

  private static setTokens(token: string, refreshToken: string): void {
    if (typeof window === 'undefined') return;

    // 存储到统一键名
    setAccessToken(token);
    setRefreshToken(refreshToken);
    
    // 同时设置 cookie 以配合中间件
    document.cookie = `auth-token=${token}; path=/; max-age=${3600}; SameSite=Lax`;
    document.cookie = `refresh-token=${refreshToken}; path=/; max-age=${86400}; SameSite=Lax`;
  }

  private static clearTokens(): void {
    if (typeof window === 'undefined') return;

    // 清除统一键名（包含历史键名）
    clearAuthStorage();

    // 清除 cookies
    document.cookie = 'auth-token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
    document.cookie = 'refresh-token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
  }

  private static generateMockToken(user?: { id: number; username: string; role: string }): string {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({
      user_id: user?.id || 1,
      username: user?.username || 'mock_user',
      role: user?.role || 'user',
      exp: Math.floor(Date.now() / 1000) + 3600 // 1小时后过期
    }));
    const signature = btoa('mock_signature');
    
    return `${header}.${payload}.${signature}`;
  }

  private static generateMockRefreshToken(user?: { id: number }): string {
    const payload = btoa(JSON.stringify({
      user_id: user?.id || 1,
      type: 'refresh',
      exp: Math.floor(Date.now() / 1000) + 86400 // 24小时后过期
    }));
    
    return `refresh.${payload}.mock_signature`;
  }

  private static isTokenExpired(token: string): boolean {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      return Date.now() >= payload.exp * 1000;
    } catch {
      return true;
    }
  }

  private static isValidRefreshToken(token: string): boolean {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      return payload.type === 'refresh' && Date.now() < payload.exp * 1000;
    } catch {
      return false;
    }
  }

  private static setupTokenRefresh(expiresIn: number): void {
    this.clearTokenRefresh();
    
    // 在token过期前5分钟刷新
    const refreshTime = (expiresIn - 300) * 1000;
    
    if (refreshTime > 0) {
      this.refreshTimer = setTimeout(async () => {
        try {
          await this.refreshToken();
        } catch (error) {
          console.error('Auto refresh token failed:', error);
          // 自动刷新失败，清除认证状态
          this.logout();
        }
      }, refreshTime);
    }
  }

  private static clearTokenRefresh(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  /**
   * 初始化认证服务
   */
  static async initialize(): Promise<void> {
    const token = this.getToken();
    
    if (token && !this.isTokenExpired(token)) {
      try {
        const user = await this.getCurrentUser();
        useAuthStore.getState().login(user, token);
        
        // 设置自动刷新
        const payload = JSON.parse(atob(token.split('.')[1]));
        const expiresIn = payload.exp - Math.floor(Date.now() / 1000);
        this.setupTokenRefresh(expiresIn);
      } catch (error) {
        console.error('Initialize auth failed:', error);
        this.clearTokens();
      }
    }
  }
}
