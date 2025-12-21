/**
 * 统一API配置V2 - 重构版本
 * 解决API版本控制、参数标准化、错误处理等问题
 */

// API配置常量
export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL || (typeof window === 'undefined' ? 'http://localhost:8090' : ''),
  VERSION: 'v1',
  TIMEOUT: 30000,
  RETRY_ATTEMPTS: 3,
  RETRY_DELAY: 1000,
} as const;

// API端点定义
export const API_ENDPOINTS = {
  // 认证相关
  AUTH: {
    LOGIN: '/auth/login',
    LOGOUT: '/auth/logout',
    REFRESH: '/auth/refresh',
    PROFILE: '/auth/profile',
    CHANGE_PASSWORD: '/auth/change-password',
  },
  
  // 用户管理
  USERS: {
    LIST: '/users',
    DETAIL: (id: number) => `/users/${id}`,
    CREATE: '/users',
    UPDATE: (id: number) => `/users/${id}`,
    DELETE: (id: number) => `/users/${id}`,
    BATCH_DELETE: '/users/batch-delete',
    ROLES: (id: number) => `/users/${id}/roles`,
    STATUS: (id: number) => `/users/${id}/status`,
  },
  
  // 工单管理
  TICKETS: {
    LIST: '/tickets',
    DETAIL: (id: number) => `/tickets/${id}`,
    CREATE: '/tickets',
    UPDATE: (id: number) => `/tickets/${id}`,
    DELETE: (id: number) => `/tickets/${id}`,
    BATCH_DELETE: '/tickets/batch-delete',
    ASSIGN: (id: number) => `/tickets/${id}/assign`,
    STATUS: (id: number) => `/tickets/${id}/status`,
    PRIORITY: (id: number) => `/tickets/${id}/priority`,
    COMMENTS: (id: number) => `/tickets/${id}/comments`,
    ATTACHMENTS: (id: number) => `/tickets/${id}/attachments`,
    WORKFLOW: (id: number) => `/tickets/${id}/workflow`,
    EXPORT: '/tickets/export',
    IMPORT: '/tickets/import',
    STATISTICS: '/tickets/statistics',
  },
  
  // 服务目录
  SERVICES: {
    LIST: '/services',
    DETAIL: (id: number) => `/services/${id}`,
    CREATE: '/services',
    UPDATE: (id: number) => `/services/${id}`,
    DELETE: (id: number) => `/services/${id}`,
    REQUESTS: '/service-requests',
    REQUEST_DETAIL: (id: number) => `/service-requests/${id}`,
  },
  
  // 租户管理
  TENANTS: {
    LIST: '/tenants',
    DETAIL: (id: number) => `/tenants/${id}`,
    CREATE: '/tenants',
    UPDATE: (id: number) => `/tenants/${id}`,
    DELETE: (id: number) => `/tenants/${id}`,
    STATUS: (id: number) => `/tenants/${id}/status`,
    CONFIG: (id: number) => `/tenants/${id}/config`,
  },
  
  // 系统配置
  SYSTEM: {
    CONFIG: '/system/config',
    SETTINGS: '/system/settings',
    LOGS: '/system/logs',
    BACKUP: '/system/backup',
    RESTORE: '/system/restore',
    HEALTH: '/system/health',
    METRICS: '/system/metrics',
  },
} as const;

// 统一请求参数接口
export interface BasePaginationParams {
  page?: number;
  page_size?: number;
}

export interface BaseSortParams {
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface BaseSearchParams {
  search?: string;
}

export interface StandardQueryParams extends BasePaginationParams, BaseSortParams, BaseSearchParams {
  [key: string]: any;
}

// 标准化请求参数
export const normalizeParams = {
  // 分页参数标准化
  pagination: (params: BasePaginationParams) => ({
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
  }),
  
  // 排序参数标准化
  sort: (params: BaseSortParams) => ({
    sort_by: params.sort_by ?? 'created_at',
    sort_order: params.sort_order ?? 'desc',
  }),
  
  // 搜索参数标准化
  search: (params: BaseSearchParams) => ({
    search: params.search?.trim() || undefined,
  }),
  
  // 完整参数标准化
  full: (params: StandardQueryParams) => ({
    ...normalizeParams.pagination(params),
    ...normalizeParams.sort(params),
    ...normalizeParams.search(params),
    // 过滤掉空值
    ...Object.fromEntries(
      Object.entries(params).filter(([_, value]) => 
        value !== undefined && value !== null && value !== ''
      )
    ),
  }),
} as const;

// 统一响应接口
export interface StandardApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
  timestamp: string;
  request_id?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface BatchOperationResult {
  success_count: number;
  failed_count: number;
  failed_items?: number[];
  message: string;
  errors?: string[];
}

// HTTP状态码常量
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  UNPROCESSABLE_ENTITY: 422,
  TOO_MANY_REQUESTS: 429,
  INTERNAL_SERVER_ERROR: 500,
  BAD_GATEWAY: 502,
  SERVICE_UNAVAILABLE: 503,
} as const;

// API错误码常量
export const API_ERROR_CODES = {
  SUCCESS: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  
  // 客户端错误 (1000-3999)
  PARAM_ERROR: 1001,
  VALIDATION_ERROR: 1002,
  MISSING_REQUIRED_FIELD: 1003,
  INVALID_FORMAT: 1004,
  
  AUTH_FAILED: 2001,
  TOKEN_EXPIRED: 2002,
  FORBIDDEN: 2003,
  INSUFFICIENT_PERMISSIONS: 2004,
  
  NOT_FOUND: 4004,
  RESOURCE_NOT_FOUND: 4005,
  USER_NOT_FOUND: 4006,
  TENANT_NOT_FOUND: 4007,
  
  CONFLICT: 4009,
  RESOURCE_CONFLICT: 4010,
  DUPLICATE_RESOURCE: 4011,
  
  RATE_LIMITED: 4029,
  QUOTA_EXCEEDED: 4030,
  
  // 服务端错误 (5000-5999)
  INTERNAL_ERROR: 5001,
  DATABASE_ERROR: 5002,
  EXTERNAL_SERVICE_ERROR: 5003,
  NETWORK_ERROR: 5004,
  TIMEOUT_ERROR: 5005,
} as const;

// 自定义错误类型
export class ApiError extends Error {
  constructor(
    public code: number,
    public message: string,
    public details?: string,
    public status?: number,
    public request_id?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
  
  static fromResponse(response: StandardApiResponse<unknown>, status?: number): ApiError {
    return new ApiError(
      response.code,
      response.message,
      response.data as string,
      status,
      response.request_id
    );
  }
  
  isClientError(): boolean {
    return this.code >= 1000 && this.code < 4000;
  }
  
  isServerError(): boolean {
    return this.code >= 5000 && this.code < 6000;
  }
  
  isRetryable(): boolean {
    return this.code === API_ERROR_CODES.NETWORK_ERROR ||
           this.code === API_ERROR_CODES.TIMEOUT_ERROR ||
           this.code === API_ERROR_CODES.EXTERNAL_SERVICE_ERROR;
  }
}

// 请求配置接口
export interface RequestConfig extends RequestInit {
  params?: Record<string, any>;
  retries?: number;
  timeout?: number;
  skipAuth?: boolean;
  skipErrorHandler?: boolean;
}

// API客户端类
export class UnifiedApiClient {
  private baseURL: string;
  private timeout: number;
  private defaultHeaders: Record<string, string>;
  
  constructor(config: Partial<typeof API_CONFIG> = {}) {
    this.baseURL = config.BASE_URL || API_CONFIG.BASE_URL;
    this.timeout = config.TIMEOUT || API_CONFIG.TIMEOUT;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
  }
  
  // 构建完整URL
  private buildURL(endpoint: string, params?: Record<string, any>): string {
    const url = new URL(endpoint, this.baseURL);
    
    if (params) {
      const normalizedParams = normalizeParams.full(params);
      Object.entries(normalizedParams).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value));
        }
      });
    }
    
    return url.toString();
  }
  
  // 创建AbortController用于超时控制
  private createTimeoutController(): AbortController {
    const controller = new AbortController();
    setTimeout(() => controller.abort(), this.timeout);
    return controller;
  }
  
  // 获取认证token
  private getAuthHeader(): string | undefined {
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('itsm_token');
      if (token) {
        return `Bearer ${token}`;
      }
    }
    return undefined;
  }
  
  // 核心请求方法
  private async request<T = any>(
    method: string,
    endpoint: string,
    config: RequestConfig = {}
  ): Promise<T> {
    const {
      params,
      retries = API_CONFIG.RETRY_ATTEMPTS,
      timeout = this.timeout,
      skipAuth = false,
      skipErrorHandler = false,
      ...fetchConfig
    } = config;
    
    let lastError: Error;
    
    // 构建URL
    const url = this.buildURL(endpoint, params);
    
    // 构建请求头
    const headers = new Headers(this.defaultHeaders);
    
    if (!skipAuth) {
      const authHeader = this.getAuthHeader();
      if (authHeader) {
        headers.set('Authorization', authHeader);
      }
    }
    
    // 添加自定义头
    if (fetchConfig.headers) {
      for (const [key, value] of Object.entries(fetchConfig.headers)) {
        if (value !== undefined) {
          headers.set(key, String(value));
        }
      }
    }
    
    // 重试逻辑
    for (let attempt = 1; attempt <= retries; attempt++) {
      try {
        const controller = this.createTimeoutController();
        
        const response = await fetch(url, {
          method,
          headers,
          signal: controller.signal,
          ...fetchConfig,
        });
        
        // 检查响应状态
        if (!response.ok) {
          const errorData = await response.json().catch(() => null);
          const apiError = new ApiError(
            errorData?.code || response.status,
            errorData?.message || response.statusText,
            errorData?.data,
            response.status,
            errorData?.request_id
          );
          
          // 如果不可重试或已达到最大重试次数，抛出错误
          if (!apiError.isRetryable() || attempt === retries) {
            throw apiError;
          }
          
          lastError = apiError;
          // 等待后重试
          await new Promise(resolve => 
            setTimeout(resolve, API_CONFIG.RETRY_DELAY * attempt)
          );
          continue;
        }
        
        // 解析响应
        const data = await response.json();
        
        // 检查API响应码
        if (data.code !== API_ERROR_CODES.SUCCESS && 
            data.code !== API_ERROR_CODES.CREATED && 
            data.code !== API_ERROR_CODES.NO_CONTENT) {
          
          const apiError = ApiError.fromResponse(data, response.status);
          
          // 如果不可重试或已达到最大重试次数，抛出错误
          if (!apiError.isRetryable() || attempt === retries) {
            throw apiError;
          }
          
          lastError = apiError;
          await new Promise(resolve => 
            setTimeout(resolve, API_CONFIG.RETRY_DELAY * attempt)
          );
          continue;
        }
        
        return data.data as T;
        
      } catch (error) {
        lastError = error as Error;
        
        // 如果是网络错误或超时，可以重试
        if (error instanceof Error && 
            (error.name === 'AbortError' || 
             error.message.includes('fetch') ||
             error.message.includes('network'))) {
          
          if (attempt === retries) {
            throw new ApiError(
              API_ERROR_CODES.NETWORK_ERROR,
              '网络请求失败',
              error.message
            );
          }
          
          await new Promise(resolve => 
            setTimeout(resolve, API_CONFIG.RETRY_DELAY * attempt)
          );
          continue;
        }
        
        // 其他错误直接抛出
        throw error;
      }
    }
    
    throw lastError!;
  }
  
  // HTTP方法封装
  async get<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.request<T>('GET', endpoint, config);
  }
  
  async post<T>(endpoint: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.request<T>('POST', endpoint, {
      ...config,
      body: data ? JSON.stringify(data) : undefined,
    });
  }
  
  async put<T>(endpoint: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.request<T>('PUT', endpoint, {
      ...config,
      body: data ? JSON.stringify(data) : undefined,
    });
  }
  
  async patch<T>(endpoint: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.request<T>('PATCH', endpoint, {
      ...config,
      body: data ? JSON.stringify(data) : undefined,
    });
  }
  
  async delete<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.request<T>('DELETE', endpoint, config);
  }
  
  // 文件上传
  async upload<T>(endpoint: string, file: File, config?: RequestConfig): Promise<T> {
    const formData = new FormData();
    formData.append('file', file);
    
    const headers = new Headers();
    if (!config?.skipAuth) {
      const authHeader = this.getAuthHeader();
      if (authHeader) {
        headers.set('Authorization', authHeader);
      }
    }
    
    return this.request<T>('POST', endpoint, {
      ...config,
      body: formData,
      headers: Object.fromEntries(headers.entries()),
    });
  }
  
  // 批量操作
  async batch<T>(endpoint: string, data: any[], config?: RequestConfig): Promise<T> {
    return this.post<T>(endpoint, { items: data }, config);
  }
}

// 创建默认API客户端实例
export const apiClient = new UnifiedApiClient();

// 便捷方法：创建特定类型的API客户端
export const createApiClient = (config?: Partial<typeof API_CONFIG>) => {
  return new UnifiedApiClient(config);
};

// 导出类型和实例
export { ApiError };
export type { RequestConfig, StandardApiResponse, PaginatedResponse, BatchOperationResult };