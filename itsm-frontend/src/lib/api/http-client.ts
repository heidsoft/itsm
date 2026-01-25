import { API_BASE_URL } from '@/app/lib/api-config';
import { security } from '@/lib/security';
import { logger } from '@/lib/env';

// 递归将对象的 key 从 snake_case 转换为 camelCase
const toCamelCase = (obj: any): any => {
  if (Array.isArray(obj)) {
    return obj.map(v => toCamelCase(v));
  } else if (obj !== null && obj.constructor === Object) {
    return Object.keys(obj).reduce(
      (result, key) => {
        const camelKey = key.replace(/_([a-z])/g, (g) => g[1].toUpperCase());
        result[camelKey] = toCamelCase(obj[key]);
        return result;
      },
      {} as any
    );
  }
  return obj;
};

// 递归将对象的 key 从 camelCase 转换为 snake_case
const toSnakeCase = (obj: any): any => {
  if (Array.isArray(obj)) {
    return obj.map(v => toSnakeCase(v));
  } else if (obj !== null && obj.constructor === Object) {
    return Object.keys(obj).reduce(
      (result, key) => {
        const snakeKey = key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
        result[snakeKey] = toSnakeCase(obj[key]);
        return result;
      },
      {} as any
    );
  }
  return obj;
};

// Request configuration interface
export interface RequestConfig {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Record<string, string>;
  body?: BodyInit | null;
  timeout?: number;
}

// Axios-like request config used by some legacy API modules
export interface AxiosLikeRequestConfig {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  url: string;
  params?: object;
  data?: unknown;
  headers?: Record<string, string>;
  responseType?: 'json' | 'blob';
}

// API response interface
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

class HttpClient {
  private baseURL: string;
  private token: string | null = null;
  private tenantId: number | null = null;
  private tenantCode: string | null = null;
  private readonly timeout: number;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = process.env.NEXT_PUBLIC_API_URL || baseURL;
    this.timeout = parseInt(process.env.NEXT_PUBLIC_API_TIMEOUT || '30000');
    // Get token and tenant ID from localStorage
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('access_token'); // Changed to access_token
      const storedTenantId = localStorage.getItem('current_tenant_id');
      this.tenantId = storedTenantId ? parseInt(storedTenantId) : null;
      this.tenantCode = localStorage.getItem('current_tenant_code') || null;
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('access_token', token); // Changed to access_token
      // 同步更新 Cookie，有效期 15 分钟
      document.cookie = `auth-token=${token}; path=/; max-age=900; SameSite=Lax`;
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('access_token'); // Changed to access_token
      // 清除 Cookie
      document.cookie = 'auth-token=; path=/; expires=Thu, 01 Jan 1970 00:00:01 GMT;';
      document.cookie = 'refresh-token=; path=/; expires=Thu, 01 Jan 1970 00:00:01 GMT;';
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
    logger.info('HttpClient.setTenantCode:', code);
    if (typeof window !== 'undefined') {
      if (code) {
        localStorage.setItem('current_tenant_code', code);
      } else {
        localStorage.removeItem('current_tenant_code');
      }
    }
  }

  // Get tenant code
  getTenantCode(): string | null {
    return this.tenantCode;
  }

  getTenantId(): number | null {
    return this.tenantId;
  }

  getAuthToken(): string | null {
    return this.token;
  }

  // Backward-compat helpers (some legacy code expects these)
  getToken(): string | null {
    return this.getAuthToken();
  }

  getBaseURL(): string {
    return this.baseURL;
  }

  private getHeaders(): Record<string, string> {
    // Set secure request headers
    const csrfToken = security.csrf.getTokenFromMeta();
    const headers: Record<string, string> = {
      ...security.network.getSecureHeaders(csrfToken || undefined),
    };

    // Dynamically get the latest token and tenantId
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

  // Independent token refresh method to avoid circular dependencies
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
          // Update access token
          this.setToken(data.data.access_token);
          // Also update instance variable
          this.token = data.data.access_token;
          return true;
        }
      }
      return false;
    } catch (error) {
      logger.error('Token refresh failed:', error);
      return false;
    }
  }

  // Core request method using fetch API (internal)
  private async requestInternal<T>(endpoint: string, config: RequestConfig): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers = this.getHeaders();
    const sanitizedHeaders = { ...headers };
    if (sanitizedHeaders.Authorization) {
      sanitizedHeaders.Authorization = '[REDACTED]';
    }
    const requestConfig: RequestInit = {
      method: config.method,
      headers: {
        ...headers,
        ...config.headers,
      },
      // 自动转换请求 body key 为 snake_case (如果是 JSON 对象)
      body: config.body && typeof config.body === 'string' && config.body.startsWith('{')
        ? JSON.stringify(toSnakeCase(JSON.parse(config.body as string)))
        : config.body,
    };

    logger.debug('HTTP Client Request:', {
      url,
      method: config.method,
      headers: sanitizedHeaders,
      body: config.body
    });

    // 在开发模式下，如果后端服务不可用，使用模拟数据
    if (process.env.NODE_ENV === 'development' && this.baseURL.includes('localhost')) {
      logger.warn('开发模式：正在连接到后端服务，如果后端服务未运行，将显示错误');
    }

    try {
      const controller = new AbortController();
      const timeoutMs = config.timeout ?? this.timeout;
      const timeoutId = setTimeout(() => controller.abort(), timeoutMs);
      
      const response = await fetch(url, {
        ...requestConfig,
        signal: controller.signal,
      });
      
      clearTimeout(timeoutId);
      
      logger.debug('HTTP Client Response:', {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers ? Object.fromEntries(response.headers.entries()) : {}
      });
      
      // If 401 error, try to refresh token
      if (response.status === 401) {
        const refreshSuccess = await this.refreshTokenInternal();
        if (refreshSuccess) {
          // Retry original request
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
          const retryData = await retryResponse.json() as ApiResponse<T>;
          logger.debug('HTTP Client Retry Response Data:', retryData);
          
          // Check response code
          if (retryData.code !== 0) {
            const rid = retryResponse.headers.get('X-Request-Id') || '';
            const suffix = rid ? ` [RID: ${rid}]` : '';
            throw new Error((retryData.message || 'Request failed') + suffix);
          }
          
          return retryData.data;
        } else {
          // Refresh failed, clear token and redirect to login
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

      const responseData = await response.json() as ApiResponse<T>;
      logger.debug('HTTP Client Raw Response Data:', responseData);
      
      // Check response code
      if (responseData.code !== 0) {
        const rid = (response.headers && response.headers.get('X-Request-Id')) || '';
        const suffix = rid ? ` [RID: ${rid}]` : '';
        throw new Error((responseData.message || 'Request failed') + suffix);
      }
      
      // 自动转换响应数据 key 为 camelCase
      return toCamelCase(responseData.data);
    } catch (error: unknown) {
      logger.error('Request failed:', error);
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new Error('请求超时，请稍后重试');
        }
        if (error.message.includes('fetch')) {
          throw new Error('无法连接到服务器，请检查网络连接和后端服务是否运行');
        }
        throw error;
      }
      throw new Error('未知错误发生');
    }
  }

  /**
   * Unified request method.
   * Supports:
   * - request('/path', { method, headers, body })
   * - request({ method, url, data, headers, responseType })
   */
  async request<T = any>(endpoint: string, config?: Partial<RequestConfig>): Promise<T>;
  async request<T = any>(config: AxiosLikeRequestConfig): Promise<T>;
  async request<T = any>(
    arg1: string | AxiosLikeRequestConfig,
    arg2?: Partial<RequestConfig>
  ): Promise<T> {
    // Axios-like style: request({ url, method, data, headers, responseType })
    if (typeof arg1 === 'object' && arg1 && 'url' in arg1) {
      const cfg = arg1 as AxiosLikeRequestConfig;
      const method = cfg.method || 'GET';
      let endpoint = cfg.url;
      if (cfg.params) {
        const qs = new URLSearchParams();
        Object.entries(cfg.params as Record<string, unknown>).forEach(([k, v]) => {
          if (v !== undefined && v !== null) qs.append(k, String(v));
        });
        const q = qs.toString();
        if (q) endpoint += (endpoint.includes('?') ? '&' : '?') + q;
      }

      let body: BodyInit | null | undefined = undefined;
      if (cfg.data !== undefined) {
        if (cfg.data instanceof FormData) body = cfg.data;
        else body = JSON.stringify(cfg.data);
      }

      const data =
        cfg.responseType === 'blob'
          ? // blob passthrough
            await (async () => {
              const url = `${this.baseURL}${endpoint}`;
              const headers = { ...this.getHeaders(), ...(cfg.headers || {}) };
              if (body instanceof FormData) delete (headers as any)['Content-Type'];
              else headers['Content-Type'] = headers['Content-Type'] || 'application/json';
              const res = await fetch(url, { method, headers, body: body as any });
              if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
              return (await res.blob()) as any;
            })()
          : await this.requestInternal<T>(endpoint, {
              method,
              headers: cfg.headers,
              body: body as any,
            });

      return data as T;
    }

    // Fetch-like style: request('/path', { ... })
    const endpoint = arg1 as string;
    const cfg = arg2 || {};
    return this.requestInternal<T>(endpoint, {
      method: cfg.method || 'GET',
      headers: cfg.headers,
      body: cfg.body as any,
      timeout: cfg.timeout,
    });
  }

  async get<T>(endpoint: string, params?: object): Promise<T> {
    let url = endpoint;
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params as Record<string, unknown>).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.append(key, String(value));
        }
      });
      url += `?${searchParams.toString()}`;
    }
    
    return this.requestInternal<T>(url, {
      method: 'GET',
    });
  }

  async post<T = any>(
    endpoint: string,
    data?: unknown,
    config?: {
      onUploadProgress?: (progress: number) => void;
      headers?: Record<string, string>;
      responseType?: 'json' | 'blob';
    }
  ): Promise<T> {
    // 如果是 FormData，直接传递，不进行 JSON.stringify
    if (data instanceof FormData) {
      const url = `${this.baseURL}${endpoint}`;
      const headers = this.getHeaders();
      // FormData 上传时，不要设置 Content-Type，让浏览器自动设置（包含 boundary）
      delete headers['Content-Type'];
      
      // 如果支持上传进度，使用 XMLHttpRequest
      if (config?.onUploadProgress) {
        return new Promise<T>((resolve, reject) => {
          const xhr = new XMLHttpRequest();
          
          xhr.upload.addEventListener('progress', (event) => {
            if (event.lengthComputable && config.onUploadProgress) {
              const progress = Math.round((event.loaded * 100) / event.total);
              config.onUploadProgress(progress);
            }
          });
          
          xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
              try {
                const response = JSON.parse(xhr.responseText) as ApiResponse<T>;
                if (response.code === 0) {
                  resolve(response.data);
                } else {
                  reject(new Error(response.message || 'Request failed'));
                }
              } catch (error) {
                reject(new Error('Failed to parse response'));
              }
            } else {
              reject(new Error(`HTTP error! status: ${xhr.status}`));
            }
          });
          
          xhr.addEventListener('error', () => {
            reject(new Error('Network error'));
          });
          
          xhr.open('POST', url);
          Object.entries(headers).forEach(([key, value]) => {
            if (key !== 'Content-Type') {
              xhr.setRequestHeader(key, value);
            }
          });
          xhr.send(data);
        });
      }
      
      // 不支持进度时，使用 fetch
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          ...headers,
          ...(config?.headers || {}),
        },
        body: data,
      });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      if (config?.responseType === 'blob') {
        return (await response.blob()) as any;
      }

      const responseData = (await response.json()) as ApiResponse<T>;
      if (responseData.code !== 0) {
        throw new Error(responseData.message || 'Request failed');
      }
      
      return responseData.data;
    }
    
    return this.requestInternal<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      headers: config?.headers,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T = any>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'DELETE',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  // 分页查询方法
  async getPaginated<T>(
    endpoint: string, 
    params?: {
      page?: number;
      pageSize?: number;
      sortBy?: string;
      sortOrder?: 'asc' | 'desc';
      filters?: Record<string, any>;
    }
  ): Promise<{
    data: T[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (key === 'filters' && typeof value === 'object') {
            Object.entries(value).forEach(([filterKey, filterValue]) => {
              if (filterValue !== undefined && filterValue !== null) {
                queryParams.append(`filters[${filterKey}]`, String(filterValue));
              }
            });
          } else {
            queryParams.append(key, String(value));
          }
        }
      });
    }
    
    const url = queryParams.toString() 
      ? `${endpoint}?${queryParams.toString()}` 
      : endpoint;
      
    return this.get(url);
  }

  // 批量操作方法
  async batchOperation<T>(
    endpoint: string, 
    operation: string, 
    data: any[]
  ): Promise<T[]> {
    return this.post<T[]>(endpoint, {
      operation,
      data
    });
  }

  // 文件上传方法
  async uploadFile(
    endpoint: string, 
    file: File, 
    onProgress?: (progress: number) => void
  ): Promise<{
    url: string;
    filename: string;
    size: number;
  }> {
    const formData = new FormData();
    formData.append('file', file);

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable && onProgress) {
          const progress = (event.loaded / event.total) * 100;
          onProgress(progress);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const response = JSON.parse(xhr.responseText);
            resolve(response);
          } catch (error) {
            reject(new Error('Invalid response format'));
          }
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      const uploadUrl = endpoint.startsWith('http') ? endpoint : `${this.baseURL}${endpoint}`;
      xhr.open('POST', uploadUrl);
      
      // 添加认证头
      const token = this.getAuthToken();
      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }

      xhr.send(formData);
    });
  }
}

export const httpClient = new HttpClient();
export default HttpClient;
