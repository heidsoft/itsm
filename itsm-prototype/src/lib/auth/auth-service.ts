import { httpClient } from '../http-client';
import { useAuthStore, User } from '../store/auth-store';

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

// 刷新token请求接口
export interface RefreshTokenRequest {
  refresh_token: string;
}

// 刷新token响应接口
export interface RefreshTokenResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
}

// 修改密码请求接口
export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

/**
 * 认证服务类
 * 处理用户认证相关的所有操作
 */
export class AuthService {
  private static readonly TOKEN_KEY = 'auth_token';
  private static readonly REFRESH_TOKEN_KEY = 'refresh_token';
  private static refreshTimer: NodeJS.Timeout | null = null;

  /**
   * 用户登录
   * @param credentials 登录凭据
   * @returns 登录结果
   */
  static async login(credentials: LoginRequest): Promise<LoginResponse> {
    try {
      const response = await httpClient.post<LoginResponse>('/api/v1/auth/login', credentials);
      
      // 保存认证信息
      this.setTokens(response.token, response.refresh_token);
      
      // 更新认证状态
      useAuthStore.getState().login(response.user, response.token);
      
      // 设置自动刷新token
      this.setupTokenRefresh(response.expires_in);
      
      return response;
    } catch (error) {
      console.error('AuthService.login error:', error);
      throw error;
    }
  }

  /**
   * 用户登出
   */
  static async logout(): Promise<void> {
    try {
      // 调用后端登出接口
      await httpClient.post('/api/v1/auth/logout');
    } catch (error) {
      console.error('AuthService.logout error:', error);
      // 即使后端登出失败，也要清除本地状态
    } finally {
      // 清除本地认证信息
      this.clearTokens();
      
      // 清除认证状态
      useAuthStore.getState().logout();
      
      // 清除自动刷新定时器
      this.clearTokenRefresh();
    }
  }

  /**
   * 刷新访问token
   * @returns 新的token信息
   */
  static async refreshToken(): Promise<RefreshTokenResponse> {
    try {
      const refreshToken = this.getRefreshToken();
      if (!refreshToken) {
        throw new Error('No refresh token available');
      }

      const response = await httpClient.post<RefreshTokenResponse>('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      });

      // 更新token
      this.setTokens(response.token, response.refresh_token);
      
      // 更新store中的token
      const authStore = useAuthStore.getState();
      if (authStore.user) {
        authStore.login(authStore.user, response.token);
      }

      // 重新设置自动刷新
      this.setupTokenRefresh(response.expires_in);

      return response;
    } catch (error) {
      console.error('AuthService.refreshToken error:', error);
      // 刷新失败，清除认证状态
      this.logout();
      throw error;
    }
  }

  /**
   * 获取当前用户信息
   * @returns 用户信息
   */
  static async getCurrentUser(): Promise<User> {
    try {
      const response = await httpClient.get<{ user: User }>('/api/v1/auth/me');
      
      // 更新store中的用户信息
      useAuthStore.getState().updateUser(response.user);
      
      return response.user;
    } catch (error) {
      console.error('AuthService.getCurrentUser error:', error);
      throw error;
    }
  }

  /**
   * 修改密码
   * @param data 密码修改数据
   */
  static async changePassword(data: ChangePasswordRequest): Promise<void> {
    try {
      await httpClient.post('/api/v1/auth/change-password', data);
    } catch (error) {
      console.error('AuthService.changePassword error:', error);
      throw error;
    }
  }

  /**
   * 检查认证状态
   * @returns 是否已认证
   */
  static isAuthenticated(): boolean {
    const token = this.getToken();
    return !!token && !this.isTokenExpired(token);
  }

  /**
   * 获取访问token
   * @returns 访问token
   */
  static getToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(this.TOKEN_KEY);
  }

  /**
   * 获取刷新token
   * @returns 刷新token
   */
  static getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  /**
   * 设置tokens
   * @param token 访问token
   * @param refreshToken 刷新token
   */
  private static setTokens(token: string, refreshToken: string): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(this.TOKEN_KEY, token);
    localStorage.setItem(this.REFRESH_TOKEN_KEY, refreshToken);
  }

  /**
   * 清除tokens
   */
  private static clearTokens(): void {
    if (typeof window === 'undefined') return;
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.REFRESH_TOKEN_KEY);
  }

  /**
   * 检查token是否过期
   * @param token JWT token
   * @returns 是否过期
   */
  private static isTokenExpired(token: string): boolean {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      const currentTime = Math.floor(Date.now() / 1000);
      return payload.exp < currentTime;
    } catch {
      return true;
    }
  }

  /**
   * 设置自动刷新token
   * @param expiresIn token过期时间（秒）
   */
  private static setupTokenRefresh(expiresIn: number): void {
    this.clearTokenRefresh();
    
    // 在token过期前5分钟刷新
    const refreshTime = (expiresIn - 300) * 1000;
    
    if (refreshTime > 0) {
      this.refreshTimer = setTimeout(() => {
        this.refreshToken().catch((error) => {
          console.error('Auto refresh token failed:', error);
        });
      }, refreshTime);
    }
  }

  /**
   * 清除自动刷新定时器
   */
  private static clearTokenRefresh(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  /**
   * 初始化认证状态
   * 应用启动时调用，恢复认证状态
   */
  static async initialize(): Promise<void> {
    try {
      const token = this.getToken();
      if (!token || this.isTokenExpired(token)) {
        // token无效或过期，尝试刷新
        await this.refreshToken();
      } else {
        // token有效，获取用户信息
        await this.getCurrentUser();
      }
    } catch (error) {
      console.error('AuthService.initialize error:', error);
      // 初始化失败，清除认证状态
      this.logout();
    }
  }
}