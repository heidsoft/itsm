import { API_BASE_URL, ApiResponse } from './api-config';

interface RequestConfig {
  method: string;
  body?: string;
  headers?: Record<string, string>;
}

class HttpClient {
  private baseURL: string;
  private token: string | null = null;
  private tenantId: number | null = null;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
    // 从localStorage获取token和租户ID
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('access_token'); // 改为 access_token
      const storedTenantId = localStorage.getItem('current_tenant_id');
      this.tenantId = storedTenantId ? parseInt(storedTenantId) : null;
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

  getTenantId(): number | null {
    return this.tenantId;
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    if (this.tenantId) {
      headers['X-Tenant-ID'] = this.tenantId.toString();
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
      const response = await fetch(`${this.baseURL}/api/refresh-token`, {
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
  private async request<T>(endpoint: string, config: RequestConfig): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    const requestConfig: RequestInit = {
      method: config.method,
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
      body: config.body,
    };

    try {
      const response = await fetch(url, requestConfig);
      
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
            throw new Error(`HTTP error! status: ${retryResponse.status}`);
          }
          return await retryResponse.json();
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
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error: any) {
      console.error('Request failed:', error);
      throw error;
    }
  }

  async get<T>(endpoint: string, params?: Record<string, any>): Promise<ApiResponse<T>> {
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

  async post<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'DELETE',
    });
  }
}

export const httpClient = new HttpClient();
export default HttpClient;