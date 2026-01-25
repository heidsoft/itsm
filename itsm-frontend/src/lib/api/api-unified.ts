/**
 * 统一API配置
 *
 * @deprecated 此文件已弃用，请使用以下替代方案:
 * - 类型定义: @/lib/api/types
 * - API端点: @/lib/api/types (API_URLS)
 * - HTTP客户端: @/lib/api/http-client
 *
 * 此文件将在后续版本中删除。
 */

import commonTypes from '../../../shared-types/common-types.json';

// API配置类型
export interface APIConfig {
  baseURL: string;
  version: string;
  endpoints: Record<string, string>;
  timeout: number;
  retryCount: number;
}

// 统一API配置
export const API_CONFIG: APIConfig = {
  baseURL: commonTypes.api.baseURL,
  version: commonTypes.api.version,
  endpoints: commonTypes.api.endpoints,
  timeout: 30000, // 30秒
  retryCount: 3,
};

// 端点URL构建器
export class APIEndpointBuilder {
  private config: APIConfig;

  constructor(config: APIConfig = API_CONFIG) {
    this.config = config;
  }

  // 获取完整端点URL
  endpoint(key: string, params?: Record<string, string | number>): string {
    const baseEndpoint = this.config.endpoints[key];
    if (!baseEndpoint) {
      throw new Error(`Unknown API endpoint: ${key}`);
    }

    let endpoint = `${this.config.baseURL}${baseEndpoint}`;
    
    if (params) {
      Object.entries(params).forEach(([param, value]) => {
        endpoint = endpoint.replace(`:${param}`, String(value));
      });
    }

    return endpoint;
  }

  // 获取资源详情URL
  resource(key: string, id: string | number, action?: string): string {
    const resourceEndpoint = this.endpoint(key);
    const url = `${resourceEndpoint}/${id}`;
    return action ? `${url}/${action}` : url;
  }

  // 获取集合URL
  collection(key: string): string {
    return this.endpoint(key);
  }

  // 构建查询字符串
  buildQuery<T = string>(params: Record<string, T | T[] | undefined | null>): string {
    const searchParams = new URLSearchParams();

    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        if (Array.isArray(value)) {
          value.forEach(item => searchParams.append(key, String(item)));
        } else {
          searchParams.append(key, String(value));
        }
      }
    });

    const queryString = searchParams.toString();
    return queryString ? `?${queryString}` : '';
  }

  // 完整URL构建器
  url(key: string, params?: Record<string, string | number>, queryParams?: Record<string, unknown>): string {
    const baseUrl = this.endpoint(key, params);
    return queryParams ? `${baseUrl}${this.buildQuery(queryParams)}` : baseUrl;
  }
}

// 创建全局端点构建器实例
export const apiEndpoints = new APIEndpointBuilder();

// 便捷方法
export const API_URLS = {
  // 认证相关
  LOGIN: () => apiEndpoints.collection('auth') + '/login',
  LOGOUT: () => apiEndpoints.collection('auth') + '/logout',
  REFRESH: () => apiEndpoints.collection('auth') + '/refresh',
  PROFILE: () => apiEndpoints.collection('auth') + '/profile',
  
  // 用户管理
  USERS: () => apiEndpoints.collection('users'),
  USER: (id: string | number) => apiEndpoints.resource('users', id),
  
  // 事件管理
  INCIDENTS: () => apiEndpoints.collection('incidents'),
  INCIDENT: (id: string | number) => apiEndpoints.resource('incidents', id),
  INCIDENT_COMMENTS: (id: string | number) => apiEndpoints.resource('incidents', id, 'comments'),
  INCIDENT_ATTACHMENTS: (id: string | number) => apiEndpoints.resource('incidents', id, 'attachments'),
  
  // 变更管理
  CHANGES: () => apiEndpoints.collection('changes'),
  CHANGE: (id: string | number) => apiEndpoints.resource('changes', id),
  CHANGE_APPROVALS: (id: string | number) => apiEndpoints.resource('changes', id, 'approvals'),
  
  // 服务管理
  SERVICES: () => apiEndpoints.collection('services'),
  SERVICE: (id: string | number) => apiEndpoints.resource('services', id),
  
  // 仪表板
  DASHBOARD: () => apiEndpoints.collection('dashboard'),
  DASHBOARD_WIDGETS: () => apiEndpoints.collection('dashboard') + '/widgets',
  
  // SLA管理
  SLA_RULES: () => apiEndpoints.collection('sla'),
  SLA_METRICS: () => apiEndpoints.collection('sla') + '/metrics',
  SLA_VIOLATIONS: () => apiEndpoints.collection('sla') + '/violations',
  
  // 报表
  REPORTS: () => apiEndpoints.collection('reports'),
  REPORT_GENERATE: () => apiEndpoints.collection('reports') + '/generate',
  
  // 知识库
  KNOWLEDGE: () => apiEndpoints.collection('knowledge'),
  ARTICLE: (id: string | number) => apiEndpoints.resource('knowledge', id),
  ARTICLE_VERSIONS: (id: string | number) => apiEndpoints.resource('knowledge', id, 'versions'),
} as const;

// 分页参数标准化
export interface StandardPaginationParams {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 标准化分页请求参数
export const normalizePaginationParams = (params: Record<string, unknown>): StandardPaginationParams => {
  return {
    page: params.page as number ?? 1,
    page_size: (params.page_size ?? params.pageSize) as number ?? 20,
    sort_by: (params.sort_by ?? params.sortBy) as string,
    sort_order: (params.sort_order ?? params.sortOrder) as 'asc' | 'desc' ?? 'desc',
  };
};

// 日期范围参数标准化
export interface DateRangeParams {
  date_from?: string;
  date_to?: string;
  created_from?: string;
  created_to?: string;
}

export const normalizeDateRangeParams = (params: Record<string, unknown>): DateRangeParams => {
  return {
    date_from: (params.date_from ?? params.dateFrom ?? params.startDate) as string,
    date_to: (params.date_to ?? params.dateTo ?? params.endDate) as string,
    created_from: params.created_from as string ?? params.createdFrom as string,
    created_to: params.created_to as string ?? params.createdTo as string,
  };
};

// 通用API响应类型
export interface StandardApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  timestamp?: string;
  trace_id?: string;
}

// 分页响应类型
export interface PaginatedApiResponse<T = unknown> {
  code: number;
  message: string;
  data: {
    items: T[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
  };
  timestamp?: string;
  trace_id?: string;
}

// 批量操作请求类型
export interface BatchRequest<T = unknown> {
  ids: (string | number)[];
  action: string;
  options?: Record<string, T>;
}

// 批量操作响应类型
export interface BatchResponse<T = unknown> {
  success: number;
  failed: number;
  errors?: Array<{
    id: string | number;
    error: string;
  }>;
  data?: T;
}

// API错误类型
export interface ApiError<T = unknown> {
  code: number;
  message: string;
  details?: Record<string, T>;
  field?: string;
  timestamp?: string;
  trace_id?: string;
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
  INTERNAL_SERVER_ERROR: 500,
  SERVICE_UNAVAILABLE: 503,
} as const;

// 业务错误码常量
export const ERROR_CODES = {
  SUCCESS: 0,
  VALIDATION_ERROR: 1001,
  AUTH_FAILED: 2001,
  PERMISSION_DENIED: 2002,
  TOKEN_EXPIRED: 2003,
  RESOURCE_NOT_FOUND: 3001,
  DUPLICATE_RESOURCE: 3002,
  OPERATION_NOT_ALLOWED: 4001,
  EXTERNAL_SERVICE_ERROR: 5001,
  INTERNAL_ERROR: 9001,
} as const;
