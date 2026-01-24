/**
 * 企业级ITSM认证API示例
 * 展示如何安全地与后端进行认证交互
 */

// 类型定义
export interface LoginRequest {
  username: string;
  password: string;
  rememberMe?: boolean;
  totpCode?: string;
  csrfToken?: string;
}

export interface LoginResponse {
  success: boolean;
  token?: string;
  refreshToken?: string;
  requiresMfa?: boolean;
  mfaType?: 'totp' | 'webauthn';
  user?: {
    id: string;
    username: string;
    email: string;
    role: string;
    permissions: string[];
  };
  error?: string;
}

export interface RefreshTokenResponse {
  success: boolean;
  token?: string;
  error?: string;
}

export interface WebAuthnChallengeResponse {
  success: boolean;
  challenge?: string;
  error?: string;
}

// API配置
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_TIMEOUT = 30000; // 30秒超时

// 统一 token 存储工具
import { setAccessToken, getAccessToken, clearAuthStorage } from '@/lib/auth/token-storage';

// API端点
const API_ENDPOINTS = {
  LOGIN: '/api/v1/auth/login',
  LOGOUT: '/api/v1/auth/logout',
  REFRESH: '/api/v1/auth/refresh',
  PROFILE: '/api/v1/auth/profile',
  VALIDATE: '/api/v1/auth/validate',
  CSRF: '/api/v1/auth/csrf',
  WEBAUTHN_CHALLENGE: '/api/v1/auth/webauthn/challenge',
  WEBAUTHN_VERIFY: '/api/v1/auth/webauthn/verify',
  SSO_INITIATE: '/api/v1/auth/sso/initiate',
} as const;

// HTTP客户端配置
class AuthApiClient {
  private baseURL: string;
  private timeout: number;

  constructor(baseURL: string = API_BASE_URL, timeout: number = API_TIMEOUT) {
    this.baseURL = baseURL;
    this.timeout = timeout;
  }

  /**
   * 获取CSRF Token
   */
  async getCsrfToken(): Promise<string> {
    try {
      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.CSRF}`, {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to get CSRF token');
      }

      const data = await response.json();
      return data.token;
    } catch (error) {
      console.error('Error getting CSRF token:', error);
      throw error;
    }
  }

  /**
   * 用户登录
   */
  async login(loginData: LoginRequest): Promise<LoginResponse> {
    try {
      // 获取CSRF Token（如果没有提供）
      const csrfToken = loginData.csrfToken || await this.getCsrfToken();

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);

      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.LOGIN}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrfToken,
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({
          username: loginData.username,
          password: loginData.password,
          remember_me: loginData.rememberMe,
          totp_code: loginData.totpCode,
        }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || 'Login failed',
        };
      }

      // 安全地存储token（仅在成功时）
      if (data.token) {
        // 统一使用 access_token（通过 token-storage 统一管理）
        setAccessToken(data.token);
      }

      return {
        success: true,
        token: data.token,
        refreshToken: data.refresh_token,
        requiresMfa: data.requires_mfa,
        mfaType: data.mfa_type,
        user: data.user,
      };
    } catch (error) {
      console.error('Login error:', error);
      
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          return {
            success: false,
            error: 'Request timeout',
          };
        }
        
        return {
          success: false,
          error: error.message,
        };
      }

      return {
        success: false,
        error: 'Network error',
      };
    }
  }

  /**
   * 刷新Token
   */
  async refreshToken(refreshToken: string): Promise<RefreshTokenResponse> {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);

      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.REFRESH}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${refreshToken}`,
        },
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || 'Token refresh failed',
        };
      }

      return {
        success: true,
        token: data.token,
      };
    } catch (error) {
      console.error('Token refresh error:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  /**
   * 用户登出
   */
  async logout(): Promise<{ success: boolean; error?: string }> {
    try {
      const token = getAccessToken();
      
      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.LOGOUT}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { 'Authorization': `Bearer ${token}` }),
        },
      });

      // 清除本地存储的token（统一清理，包含历史键名）
      clearAuthStorage();

      if (!response.ok) {
        const data = await response.json();
        return {
          success: false,
          error: data.message || 'Logout failed',
        };
      }

      return { success: true };
    } catch (error) {
      console.error('Logout error:', error);
      
      // 即使请求失败，也要清除本地token（统一清理，包含历史键名）
      clearAuthStorage();
      
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  /**
   * WebAuthn认证挑战
   */
  async getWebAuthnChallenge(username: string): Promise<WebAuthnChallengeResponse> {
    try {
      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.WEBAUTHN_CHALLENGE}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username }),
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || 'Failed to get WebAuthn challenge',
        };
      }

      return {
        success: true,
        challenge: data.challenge,
      };
    } catch (error) {
      console.error('WebAuthn challenge error:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  /**
   * WebAuthn认证验证
   */
  async verifyWebAuthn(credential: WebAuthnCredential): Promise<LoginResponse> {
    try {
      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.WEBAUTHN_VERIFY}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ credential }),
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || 'WebAuthn verification failed',
        };
      }

      return {
        success: true,
        token: data.token,
        refreshToken: data.refresh_token,
        user: data.user,
      };
    } catch (error) {
      console.error('WebAuthn verification error:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  /**
   * SSO登录重定向
   */
  async initiateSSOLogin(provider: string = 'default'): Promise<{ success: boolean; redirectUrl?: string; error?: string }> {
    try {
      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.SSO_INITIATE}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ provider }),
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || 'SSO initiation failed',
        };
      }

      return {
        success: true,
        redirectUrl: data.redirect_url,
      };
    } catch (error) {
      console.error('SSO initiation error:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }

  /**
   * 验证当前token是否有效
   */
  async validateToken(): Promise<{ valid: boolean; user?: WebAuthnUser; error?: string }> {
    try {
      const token = getAccessToken();
      
      if (!token) {
        return { valid: false, error: 'No token found' };
      }

      const response = await fetch(`${this.baseURL}${API_ENDPOINTS.VALIDATE}`, {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        return { valid: false, error: 'Token validation failed' };
      }

      const data = await response.json();
      return {
        valid: true,
        user: data.user,
      };
    } catch (error) {
      console.error('Token validation error:', error);
      return {
        valid: false,
        error: error instanceof Error ? error.message : 'Network error',
      };
    }
  }
}

// 导出单例实例
export const authApiClient = new AuthApiClient();

// WebAuthn类型定义
export interface WebAuthnCredential {
  id: string;
  rawId: ArrayBuffer;
  response: {
    authenticatorData: ArrayBuffer;
    clientDataJSON: ArrayBuffer;
    signature: ArrayBuffer;
    userHandle?: ArrayBuffer;
  };
  type: 'public-key';
}

export interface WebAuthnUser {
  id: string;
  username: string;
  email: string;
  role: string;
  permissions: string[];
}

// 导出便捷方法
export const AuthAPI = {
  login: (data: LoginRequest) => authApiClient.login(data),
  logout: () => authApiClient.logout(),
  refreshToken: (token: string) => authApiClient.refreshToken(token),
  getCsrfToken: () => authApiClient.getCsrfToken(),
  getWebAuthnChallenge: (username: string) => authApiClient.getWebAuthnChallenge(username),
  verifyWebAuthn: (credential: WebAuthnCredential) => authApiClient.verifyWebAuthn(credential),
  initiateSSOLogin: (provider?: string) => authApiClient.initiateSSOLogin(provider),
  validateToken: () => authApiClient.validateToken(),
};

export default AuthAPI;