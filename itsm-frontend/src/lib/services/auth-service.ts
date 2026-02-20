import { API_BASE_URL, Tenant } from '@/lib/api/api-config';
import { useAuthStore } from '@/lib/store/auth-store';

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

  // Backward-compatible helpers used by some UI providers
  static getToken(): string | null {
    return this.getAccessToken();
  }

  static getCurrentUser() {
    const { user } = useAuthStore.getState();
    return user;
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
    // 首先检查localStorage中的token（这个是同步的，不依赖hydration）
    const token = this.getAccessToken();
    if (!token) {
      // 即使没有token，也要检查store的持久化状态
      const { isAuthenticated } = useAuthStore.getState();
      return isAuthenticated;
    }

    // 对于开发环境，允许mock token
    if (token.startsWith('mock_')) {
      return true;
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
    console.log('makeRequest called with:', { url, method: options.method, body: options.body });
    
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    console.log('makeRequest response status:', response.status);
    console.log('makeRequest response headers:', response.headers);

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const responseData = await response.json() as { code: number; message: string; data: T };
    console.log('makeRequest response data:', responseData);
    
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
      }>('/api/v1/refresh-token', {
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
      console.log('AuthService.login called with:', { username, password: '***', tenantCode });
      
      const data = await this.makeRequest<{
        access_token: string;
        refresh_token: string;
        user: unknown;
      }>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({
          username,
          password,
          tenant_code: tenantCode,
        }),
      });
      
      console.log('AuthService.login response data:', data);
      
      // 存储tokens
      this.setTokens(data.access_token, data.refresh_token);
      
      // 使用store管理登录状态
      const { login } = useAuthStore.getState();
      console.log('AuthService.login - 调用Zustand login方法');
      const u = data.user as any;
      login(
        {
          id: Number(u?.id || 0),
          username: String(u?.username || username),
          role: String(u?.role || 'end_user'),
          email: String(u?.email || ''),
          name: String(u?.name || u?.full_name || ''),
          tenantId: u?.tenant_id ? Number(u.tenant_id) : undefined,
          department: u?.department,
          permissions: u?.permissions,
          createdAt: u?.created_at,
          updatedAt: u?.updated_at,
        },
        data.access_token,
        { 
          id: 1, 
          name: "默认租户", 
          code: "default",
          type: 'standard',
          status: 'active',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        } as Tenant
      );
      
      // 验证状态是否正确设置
      const { isAuthenticated } = useAuthStore.getState();
      console.log('AuthService.login - 登录后Zustand状态:', { isAuthenticated });
      
      console.log('AuthService.login completed successfully');
      return true;
    } catch (error) {
      console.error('Login failed:', error);
      return false;
    }
  }

  // 注册
  static async register(params: {
    username: string;
    email: string;
    password: string;
    fullName: string;
    phone?: string;
    company?: string;
    role?: string;
  }): Promise<boolean> {
    try {
      console.log('AuthService.register called with:', { username: params.username, email: params.email });

      await this.makeRequest<{ id: number; username: string; email: string; message: string }>(
        '/api/v1/auth/register',
        {
          method: 'POST',
          body: JSON.stringify({
            username: params.username,
            email: params.email,
            password: params.password,
            full_name: params.fullName,
            phone: params.phone,
            company: params.company,
            role: params.role,
          }),
        }
      );

      console.log('AuthService.register completed successfully');
      return true;
    } catch (error) {
      console.error('Registration failed:', error);
      return false;
    }
  }

  // 发送密码重置邮件
  static async forgotPassword(email: string, tenantCode?: string): Promise<boolean> {
    try {
      console.log('AuthService.forgotPassword called with:', { email });

      await this.makeRequest<{ message: string }>('/api/v1/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({
          email,
          tenant_code: tenantCode,
        }),
      });

      console.log('AuthService.forgotPassword completed successfully');
      return true;
    } catch (error) {
      console.error('Forgot password request failed:', error);
      return false;
    }
  }

  // 重置密码
  static async resetPassword(params: {
    token: string;
    email: string;
    password: string;
    passwordConfirm: string;
  }): Promise<boolean> {
    try {
      console.log('AuthService.resetPassword called with:', { email: params.email });

      await this.makeRequest<{ message: string }>('/api/v1/auth/reset-password', {
        method: 'POST',
        body: JSON.stringify({
          token: params.token,
          email: params.email,
          password: params.password,
          password_confirm: params.passwordConfirm,
        }),
      });

      console.log('AuthService.resetPassword completed successfully');
      return true;
    } catch (error) {
      console.error('Reset password failed:', error);
      return false;
    }
  }

  // 验证重置令牌
  static async validateResetToken(token: string, email: string): Promise<boolean> {
    try {
      console.log('AuthService.validateResetToken called with:', { email });

      const result = await this.makeRequest<{ valid: boolean; email: string }>(
        '/api/v1/auth/validate-reset-token',
        {
          method: 'POST',
          body: JSON.stringify({
            token,
            email,
          }),
        }
      );

      console.log('AuthService.validateResetToken result:', result);
      return result.valid;
    } catch (error) {
      console.error('Validate reset token failed:', error);
      return false;
    }
  }
}

export default AuthService;
