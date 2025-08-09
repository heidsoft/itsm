import { API_BASE_URL } from './api-config';

interface RequestConfig {
  method: string;
  body?: string;
  headers?: Record<string, string>;
}

class HttpClient {
  private baseURL: string;
  private token: string | null = null;
  private tenantId: number | null = null;
  private tenantCode: string | null = null;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
    // 从localStorage获取token和租户ID
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('access_token'); // 改为 access_token
      const storedTenantId = localStorage.getItem('current_tenant_id');
      this.tenantId = storedTenantId ? parseInt(storedTenantId) : null;
      this.tenantCode = localStorage.getItem('current_tenant_code') || null;
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('access_token', token); // 改为 access_token
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('access_token'); // 改为 access_token
    }
  }

  setTenantId(tenantId: number | null) {
    this.tenantId = tenantId;
    if (typeof window !== 'undefined') {
      if (tenantId) {
        localStorage.setItem('current_tenant_id', tenantId.toString());
      } else {
        localStorage.removeItem('current_tenant_id');
      }
    }
  }

  setTenantCode(code: string | null) {
    this.tenantCode = code;
    if (typeof window !== 'undefined') {
      if (code) {
        localStorage.setItem('current_tenant_code', code);
      } else {
        localStorage.removeItem('current_tenant_code');
      }
    }
  }

  getTenantId(): number | null {
    return this.tenantId;
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // 动态获取最新的token和tenantId
    const currentToken = typeof window !== 'undefined' ? localStorage.getItem('access_token') : this.token;
    const currentTenantId = typeof window !== 'undefined' ? localStorage.getItem('current_tenant_id') : this.tenantId;

    if (currentToken) {
      headers['Authorization'] = `Bearer ${currentToken}`;
    }

    if (currentTenantId) {
      headers['X-Tenant-ID'] = currentTenantId.toString();
    }
    const currentTenantCode = typeof window !== 'undefined' ? localStorage.getItem('current_tenant_code') : this.tenantCode;
    if (currentTenantCode) {
      headers['X-Tenant-Code'] = currentTenantCode;
    }

    return headers;
  }

  // 刷新token的独立方法，避免循环依赖
  private async refreshTokenInternal(): Promise<boolean> {
    const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null;
    if (!refreshToken) {
      return false;
    }

    try {
      const response = await fetch(`${this.baseURL}/api/v1/refresh-token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          refresh_token: refreshToken,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        if (data.code === 0) {
          // 更新access token
          this.setToken(data.data.access_token);
          // 同时更新实例变量
          this.token = data.data.access_token;
          return true;
        }
      }
      return false;
    } catch (error) {
      console.error('Token refresh failed:', error);
      return false;
    }
  }

  // 使用 fetch API 的 request 方法
  private async request<T>(endpoint: string, config: RequestConfig): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers = this.getHeaders();
    const requestConfig: RequestInit = {
      method: config.method,
      headers: {
        ...headers,
        ...config.headers,
      },
      body: config.body,
    };

    console.log('HTTP Client Request:', {
      url,
      method: config.method,
      headers,
      body: config.body
    });

    try {
      const response = await fetch(url, requestConfig);
      
      console.log('HTTP Client Response:', {
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries())
      });
      
      // 如果是401错误，尝试刷新token
      if (response.status === 401) {
        const refreshSuccess = await this.refreshTokenInternal();
        if (refreshSuccess) {
          // 重试原请求
          const retryConfig: RequestInit = {
            ...requestConfig,
            headers: {
              ...this.getHeaders(),
              ...config.headers,
            },
          };
          const retryResponse = await fetch(url, retryConfig);
          if (!retryResponse.ok) {
            const rid = retryResponse.headers.get('X-Request-Id') || '';
            const suffix = rid ? ` [RID: ${rid}]` : '';
            throw new Error(`HTTP error! status: ${retryResponse.status}${suffix}`);
          }
          const retryData = await retryResponse.json() as { code: number; message: string; data: T };
          console.log('HTTP Client Retry Response Data:', retryData);
          
          // 检查响应码
          if (retryData.code !== 0) {
            const rid = retryResponse.headers.get('X-Request-Id') || '';
            const suffix = rid ? ` [RID: ${rid}]` : '';
            throw new Error((retryData.message || '请求失败') + suffix);
          }
          
          return retryData.data;
        } else {
          // 刷新失败，清除token并跳转到登录页
          this.clearToken();
          if (typeof window !== 'undefined') {
            localStorage.removeItem('refresh_token');
            window.location.href = '/login';
          }
          throw new Error('Authentication failed');
        }
      }

      if (!response.ok) {
        const rid = response.headers.get('X-Request-Id') || '';
        const suffix = rid ? ` [RID: ${rid}]` : '';
        throw new Error(`HTTP error! status: ${response.status}${suffix}`);
      }

      const responseData = await response.json() as { code: number; message: string; data: T };
      console.log('HTTP Client Raw Response Data:', responseData);
      
      // 检查响应码
      if (responseData.code !== 0) {
        const rid = (response.headers && response.headers.get('X-Request-Id')) || '';
        const suffix = rid ? ` [RID: ${rid}]` : '';
        throw new Error((responseData.message || '请求失败') + suffix);
      }
      
      return responseData.data;
    } catch (error: unknown) {
      console.error('Request failed:', error);
      throw error;
    }
  }

  async get<T>(endpoint: string, params?: Record<string, unknown>): Promise<T> {
    let url = endpoint;
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.append(key, String(value));
        }
      });
      url += `?${searchParams.toString()}`;
    }
    
    return this.request<T>(url, {
      method: 'GET',
    });
  }

  async post<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'DELETE',
    });
  }
}

export const httpClient = new HttpClient();
export default HttpClient;