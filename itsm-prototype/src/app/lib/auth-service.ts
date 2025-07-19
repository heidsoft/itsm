import { API_BASE_URL } from './api-config';
import { useAuthStore } from './store';

export class AuthService {
  // 设置tokens
  static setTokens(accessToken: string, refreshToken: string) {
    if (typeof window !== 'undefined') {
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
    }
  }

  // 获取access token
  static getAccessToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('access_token');
    }
    return null;
  }

  // 获取refresh token
  static getRefreshToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('refresh_token');
    }
    return null;
  }

  // 检查是否已认证
  static isAuthenticated(): boolean {
    const token = this.getAccessToken();
    if (!token) {
      return false;
    }

    try {
      // 简单检查token格式（JWT应该有3个部分）
      const parts = token.split('.');
      if (parts.length !== 3) {
        return false;
      }

      // 检查token是否过期
      const payload = JSON.parse(atob(parts[1]));
      const currentTime = Math.floor(Date.now() / 1000);
      
      if (payload.exp && payload.exp < currentTime) {
        // Token已过期，清除它
        this.clearTokens();
        return false;
      }

      return true;
    } catch (error) {
      // Token格式无效
      console.error('Invalid token format:', error);
      this.clearTokens();
      return false;
    }
  }

  // 直接使用fetch进行HTTP请求，避免循环依赖
  private static async makeRequest<T>(endpoint: string, options: RequestInit): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const responseData = await response.json() as { code: number; message: string; data: T };
    
    // 检查响应码
    if (responseData.code !== 0) {
      throw new Error(responseData.message || '请求失败');
    }
    
    return responseData.data;
  }

  // 刷新token
  static async refreshToken(): Promise<boolean> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      return false;
    }

    try {
      const data = await this.makeRequest<{
        access_token: string;
      }>('/api/refresh-token', {
        method: 'POST',
        body: JSON.stringify({
          refresh_token: refreshToken,
        }),
      });

      // 更新access token
      if (typeof window !== 'undefined') {
        localStorage.setItem('access_token', data.access_token);
      }
      return true;
    } catch (error) {
      console.error('Token refresh failed:', error);
      this.clearTokens();
      return false;
    }
  }

  // 清除所有tokens
  static clearTokens() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
    }
  }

  // 登出方法
  static logout() {
    this.clearTokens();
    const { logout } = useAuthStore.getState();
    logout();
  }

  // 修改login方法
  static async login(username: string, password: string, tenantCode?: string): Promise<boolean> {
    try {
      const data = await this.makeRequest<{
        access_token: string;
        refresh_token: string;
        user: unknown;
        tenant: unknown;
      }>('/api/login', {
        method: 'POST',
        body: JSON.stringify({
          username,
          password,
          tenant_code: tenantCode,
        }),
      });
      
      // 存储tokens
      this.setTokens(data.access_token, data.refresh_token);
      
      // 使用store管理登录状态
      const { login } = useAuthStore.getState();
      login(
        data.user,
        data.access_token,
        data.tenant
      );
      return true;
    } catch (error) {
      console.error('Login failed:', error);
      return false;
    }
  }
}

export default AuthService;