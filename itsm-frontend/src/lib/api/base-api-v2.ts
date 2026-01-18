/**
 * 增强版基础API类
 * 提供完整的CRUD操作、错误处理和类型安全
 */

import { httpClient } from './http-client';
import {
  ApiResponse,
  PaginationResponse,
  ListQueryParams,
  BatchOperationRequest,
  RequestOptions,
  ApiErrorCode,
} from './types';

/**
 * API错误类
 */
export class ApiError extends Error {
  public readonly code: number;
  public readonly originalError?: unknown;
  public readonly endpoint: string;

  constructor(
    message: string,
    code: number = ApiErrorCode.INTERNAL_ERROR,
    endpoint: string = '',
    originalError?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.endpoint = endpoint;
    this.originalError = originalError;
  }

  static fromResponse<T>(response: ApiResponse<T>, endpoint: string): ApiError {
    return new ApiError(response.message, response.code, endpoint);
  }

  static fromError(error: unknown, endpoint: string): ApiError {
    if (error instanceof ApiError) {
      return error;
    }
    if (error instanceof Error) {
      return new ApiError(error.message, ApiErrorCode.INTERNAL_ERROR, endpoint, error);
    }
    return new ApiError('Unknown error occurred', ApiErrorCode.INTERNAL_ERROR, endpoint, error);
  }

  get isAuthError(): boolean {
    return this.code === ApiErrorCode.AUTH_FAILED;
  }

  get isNotFound(): boolean {
    return this.code === ApiErrorCode.NOT_FOUND;
  }

  get isParamError(): boolean {
    return this.code >= ApiErrorCode.PARAM_ERROR && this.code < 2000;
  }
}

/**
 * API响应处理工具
 */
export class ApiResult<T> {
  private constructor(
    public readonly success: boolean,
    public readonly data: T | null,
    public readonly error: ApiError | null,
    public readonly rawResponse?: ApiResponse<T>
  ) {}

  static ok<T>(data: T, rawResponse?: ApiResponse<T>): ApiResult<T> {
    return new ApiResult(true, data, null, rawResponse);
  }

  static fail<T>(error: ApiError): ApiResult<T> {
    return new ApiResult<T>(false, null, error);
  }

  static fromResponse<T>(response: ApiResponse<T>): ApiResult<T> {
    if (response.code === ApiErrorCode.SUCCESS) {
      return ApiResult.ok(response.data, response);
    }
    return ApiResult.fail(ApiError.fromResponse(response, ''));
  }

  getOrThrow(): T {
    if (!this.success || this.data === null) {
      throw this.error;
    }
    return this.data;
  }

  getOrNull(): T | null {
    return this.data;
  }

  getOrDefault(defaultValue: T): T {
    return this.data ?? defaultValue;
  }
}

/**
 * 增强版基础API类
 */
export abstract class BaseApiV2 {
  protected static readonly basePath: string = '';

  /**
   * 获取完整API路径
   */
  protected static getPath(endpoint: string = ''): string {
    return `${this.basePath}${endpoint}`;
  }

  /**
   * 统一请求处理
   */
  protected static async request<T>(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH',
    endpoint: string,
    data?: unknown,
    options?: RequestOptions
  ): Promise<T> {
    const path = this.getPath(endpoint);
    try {
      switch (method) {
        case 'GET':
          return await httpClient.get<T>(path, data as Record<string, unknown>);
        case 'POST':
          return await httpClient.post<T>(path, data);
        case 'PUT':
          return await httpClient.put<T>(path, data);
        case 'DELETE':
          return await httpClient.delete<T>(path, data);
        case 'PATCH':
          return await httpClient.patch<T>(path, data);
        default:
          throw new Error(`Unsupported HTTP method: ${method}`);
      }
    } catch (error) {
      throw ApiError.fromError(error, path);
    }
  }

  // ==================== CRUD 操作 ====================

  /**
   * 获取列表
   */
  protected static async getList<T, P extends ListQueryParams = ListQueryParams>(
    endpoint: string,
    params?: P
  ): Promise<PaginationResponse<T>> {
    return this.request<PaginationResponse<T>>('GET', endpoint, params);
  }

  /**
   * 获取单个
   */
  protected static async getById<T>(endpoint: string, id: string | number): Promise<T> {
    return this.request<T>('GET', `${endpoint}/${id}`);
  }

  /**
   * 创建
   */
  protected static async create<T, D = Record<string, unknown>>(
    endpoint: string,
    data: D
  ): Promise<T> {
    return this.request<T>('POST', endpoint, data);
  }

  /**
   * 完全更新
   */
  protected static async update<T, D = Record<string, unknown>>(
    endpoint: string,
    id: string | number,
    data: D
  ): Promise<T> {
    return this.request<T>('PUT', `${endpoint}/${id}`, data);
  }

  /**
   * 部分更新
   */
  protected static async patch<T, D = Record<string, unknown>>(
    endpoint: string,
    id: string | number,
    data: D
  ): Promise<T> {
    return this.request<T>('PATCH', `${endpoint}/${id}`, data);
  }

  /**
   * 删除
   */
  protected static async delete<T = void>(endpoint: string, id: string | number): Promise<T> {
    return this.request<T>('DELETE', `${endpoint}/${id}`);
  }

  // ==================== 批量操作 ====================

  /**
   * 批量删除
   */
  protected static async batchDelete<T>(
    endpoint: string,
    ids: (string | number)[]
  ): Promise<T> {
    return this.request<T>('POST', `${endpoint}/batch`, { ids });
  }

  /**
   * 批量更新
   */
  protected static async batchUpdate<T, D = Record<string, unknown>>(
    endpoint: string,
    ids: (string | number)[],
    data: D
  ): Promise<T> {
    return this.request<T>('PUT', `${endpoint}/batch`, { ids, data });
  }

  // ==================== 搜索操作 ====================

  /**
   * 搜索
   */
  protected static async search<T, P extends Record<string, unknown> = Record<string, unknown>>(
    endpoint: string,
    params: P
  ): Promise<PaginationResponse<T>> {
    return this.request<PaginationResponse<T>>('GET', `${endpoint}/search`, params);
  }

  // ==================== 统计操作 ====================

  /**
   * 获取统计
   */
  protected static async getStats<T>(
    endpoint: string,
    params?: Record<string, unknown>
  ): Promise<T> {
    return this.request<T>('GET', `${endpoint}/stats`, params);
  }

  // ==================== 导出操作 ====================

  /**
   * 导出数据
   */
  protected static async export(
    endpoint: string,
    params?: Record<string, unknown>,
    format: 'csv' | 'excel' | 'pdf' = 'csv'
  ): Promise<Blob> {
    const path = this.getPath(`${endpoint}/export?format=${format}`);
    const url = `${httpClient.getBaseURL()}${path}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${httpClient.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  // ==================== 上传操作 ====================

  /**
   * 上传文件
   */
  protected static async upload<T>(
    endpoint: string,
    file: File,
    additionalData?: Record<string, string>
  ): Promise<T> {
    const formData = new FormData();
    formData.append('file', file);

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, value);
      });
    }

    return httpClient.post<T>(this.getPath(endpoint), formData);
  }

  // ==================== 工具方法 ====================

  /**
   * 构建查询参数
   */
  protected static buildQueryString(params: Record<string, unknown>): string {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        searchParams.append(key, String(value));
      }
    });
    return searchParams.toString();
  }

  /**
   * 安全解析JSON
   */
  protected static safeParse<T>(json: string, fallback: T): T {
    try {
      return JSON.parse(json);
    } catch {
      return fallback;
    }
  }
}

/**
 * 简化的CRUD API特征接口
 */
export interface CrudApiInterface<T, CreateDto, UpdateDto, QueryParams extends ListQueryParams = ListQueryParams> {
  list(params?: QueryParams): Promise<PaginationResponse<T>>;
  get(id: string | number): Promise<T>;
  create(data: CreateDto): Promise<T>;
  update(id: string | number, data: UpdateDto): Promise<T>;
  delete(id: string | number): Promise<void>;
}

/**
 * 带分页的列表API特征接口
 */
export interface PaginatedApiInterface<T, QueryParams extends ListQueryParams = ListQueryParams> {
  list(params?: QueryParams): Promise<PaginationResponse<T>>;
  search(params: QueryParams): Promise<PaginationResponse<T>>;
}

/**
 * 批量操作API特征接口
 */
export interface BatchApiInterface<T> {
  batchDelete(ids: (string | number)[]): Promise<T>;
  batchUpdate(ids: (string | number)[], data: Partial<T>): Promise<T>;
}
