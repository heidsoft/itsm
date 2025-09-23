/**
 * HTTP客户端 - 统一的API请求处理
 * 
 * 功能特性：
 * - 统一的请求/响应拦截
 * - 自动错误处理
 * - JWT Token 自动添加
 * - 请求重试机制
 * - 请求/响应日志
 */

// API响应接口
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// 请求配置接口
export interface RequestConfig {
  headers?: Record<string, string>;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

// HTTP错误类
export class HttpError extends Error {
  constructor(
    public status: number,
    public code: number,
    message: string,
    public response?: Response
  ) {
    super(message);
    this.name = 'HttpError';
  }
}

// API配置
export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
  VERSION: 'v1',
  TIMEOUT: 30000, // 30秒超时
  MAX_RETRIES: 3,
  RETRY_DELAY: 1000, // 1秒重试延迟
};

// 获取API URL
export const getApiUrl = (endpoint: string): string => {
  const baseUrl = API_CONFIG.BASE_URL.replace(/\/$/, '');
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  return `${baseUrl}/api/${API_CONFIG.VERSION}${cleanEndpoint}`;
};

// Token管理
class TokenManager {
  private static readonly TOKEN_KEY = 'auth_token';
  private static readonly REFRESH_TOKEN_KEY = 'refresh_token';

  static getToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(this.TOKEN_KEY);
  }

  static setToken(token: string): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(this.TOKEN_KEY, token);
  }

  static getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  static setRefreshToken(token: string): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(this.REFRESH_TOKEN_KEY, token);
  }

  static clearTokens(): void {
    if (typeof window === 'undefined') return;
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.REFRESH_TOKEN_KEY);
  }
}

// HTTP客户端类
class HttpClient {
  private defaultConfig: RequestConfig = {
    timeout: API_CONFIG.TIMEOUT,
    retries: API_CONFIG.MAX_RETRIES,
    retryDelay: API_CONFIG.RETRY_DELAY,
  };

  // 创建请求头
  private createHeaders(config?: RequestConfig): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...config?.headers,
    };

    // 添加认证头
    const token = TokenManager.getToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    return headers;
  }

  // 处理响应
  private async handleResponse<T>(response: Response): Promise<T> {
    const contentType = response.headers.get('content-type');
    const isJson = contentType?.includes('application/json');

    let responseData: ApiResponse<T>;
    
    try {
      if (isJson) {
        responseData = await response.json();
      } else {
        // 非JSON响应，包装为标准格式
        const text = await response.text();
        responseData = {
          code: response.ok ? 0 : response.status,
          message: response.ok ? 'success' : text || response.statusText,
          data: text as T,
        };
      }
    } catch (error) {
      throw new HttpError(
        response.status,
        response.status,
        `Failed to parse response: ${error}`,
        response
      );
    }

    // 检查HTTP状态码
    if (!response.ok) {
      throw new HttpError(
        response.status,
        responseData.code || response.status,
        responseData.message || response.statusText,
        response
      );
    }

    // 检查业务状态码
    if (responseData.code !== 0) {
      // 处理认证失败
      if (responseData.code === 2001) {
        TokenManager.clearTokens();
        // 可以在这里触发重新登录
        if (typeof window !== 'undefined') {
          window.location.href = '/login';
        }
      }
      
      throw new HttpError(
        response.status,
        responseData.code,
        responseData.message || '请求失败',
        response
      );
    }

    return responseData.data;
  }

  // 延迟函数
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // 执行请求（带重试）
  private async executeRequest<T>(
    url: string,
    options: RequestInit,
    config: RequestConfig
  ): Promise<T> {
    const maxRetries = config.retries || this.defaultConfig.retries || 0;
    let lastError: Error;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        console.log(`HTTP Request [${options.method}] ${url}`, {
          attempt: attempt + 1,
          maxRetries: maxRetries + 1,
          headers: options.headers,
        });

        const controller = new AbortController();
        const timeoutId = setTimeout(() => {
          controller.abort();
        }, config.timeout || this.defaultConfig.timeout);

        const response = await fetch(url, {
          ...options,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        console.log(`HTTP Response [${response.status}] ${url}`, {
          status: response.status,
          statusText: response.statusText,
          headers: Object.fromEntries(response.headers.entries()),
        });

        return await this.handleResponse<T>(response);
      } catch (error) {
        lastError = error as Error;
        
        console.error(`HTTP Request failed [${options.method}] ${url}`, {
          attempt: attempt + 1,
          error: lastError.message,
        });

        // 最后一次尝试，不再重试
        if (attempt === maxRetries) {
          break;
        }

        // 只对特定错误进行重试
        if (error instanceof HttpError) {
          // 4xx错误不重试（除了429）
          if (error.status >= 400 && error.status < 500 && error.status !== 429) {
            break;
          }
        }

        // 等待后重试
        const retryDelay = config.retryDelay || this.defaultConfig.retryDelay || 1000;
        await this.delay(retryDelay * Math.pow(2, attempt)); // 指数退避
      }
    }

    throw lastError!;
  }

  // GET请求
  async get<T>(endpoint: string, params?: Record<string, unknown>, config?: RequestConfig): Promise<T> {
    const url = new URL(getApiUrl(endpoint));
    
    // 添加查询参数
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value));
        }
      });
    }

    return this.executeRequest<T>(
      url.toString(),
      {
        method: 'GET',
        headers: this.createHeaders(config),
      },
      { ...this.defaultConfig, ...config }
    );
  }

  // POST请求
  async post<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.executeRequest<T>(
      getApiUrl(endpoint),
      {
        method: 'POST',
        headers: this.createHeaders(config),
        body: data ? JSON.stringify(data) : undefined,
      },
      { ...this.defaultConfig, ...config }
    );
  }

  // PUT请求
  async put<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.executeRequest<T>(
      getApiUrl(endpoint),
      {
        method: 'PUT',
        headers: this.createHeaders(config),
        body: data ? JSON.stringify(data) : undefined,
      },
      { ...this.defaultConfig, ...config }
    );
  }

  // PATCH请求
  async patch<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.executeRequest<T>(
      getApiUrl(endpoint),
      {
        method: 'PATCH',
        headers: this.createHeaders(config),
        body: data ? JSON.stringify(data) : undefined,
      },
      { ...this.defaultConfig, ...config }
    );
  }

  // DELETE请求
  async delete<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.executeRequest<T>(
      getApiUrl(endpoint),
      {
        method: 'DELETE',
        headers: this.createHeaders(config),
      },
      { ...this.defaultConfig, ...config }
    );
  }

  // 上传文件
  async upload<T>(endpoint: string, file: File, config?: RequestConfig): Promise<T> {
    const formData = new FormData();
    formData.append('file', file);

    const headers = this.createHeaders(config);
    // 删除Content-Type，让浏览器自动设置
    delete headers['Content-Type'];

    return this.executeRequest<T>(
      getApiUrl(endpoint),
      {
        method: 'POST',
        headers,
        body: formData,
      },
      { ...this.defaultConfig, ...config }
    );
  }
}

// 导出单例实例
export const httpClient = new HttpClient();

// 导出Token管理器
export { TokenManager };

// 便捷方法
export const api = {
  get: <T>(endpoint: string, params?: Record<string, unknown>, config?: RequestConfig) =>
    httpClient.get<T>(endpoint, params, config),
  
  post: <T>(endpoint: string, data?: unknown, config?: RequestConfig) =>
    httpClient.post<T>(endpoint, data, config),
  
  put: <T>(endpoint: string, data?: unknown, config?: RequestConfig) =>
    httpClient.put<T>(endpoint, data, config),
  
  patch: <T>(endpoint: string, data?: unknown, config?: RequestConfig) =>
    httpClient.patch<T>(endpoint, data, config),
  
  delete: <T>(endpoint: string, config?: RequestConfig) =>
    httpClient.delete<T>(endpoint, config),
  
  upload: <T>(endpoint: string, file: File, config?: RequestConfig) =>
    httpClient.upload<T>(endpoint, file, config),
};