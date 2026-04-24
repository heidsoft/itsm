/**
 * BaseService - 统一的 API 服务基类
 *
 * 提供标准的 CRUD 操作，减少重复代码。
 * 所有业务服务都应继承此类。
 */

import { httpClient } from '@/lib/api/http-client';

// ==================== 类型定义 ====================

/** 分页参数 */
export interface PaginationParams {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

/** 分页响应 */
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/** 列表查询参数 */
export interface ListParams extends PaginationParams {
  [key: string]: unknown;
}

/** API 错误 */
export class ApiError extends Error {
  public readonly code: number;
  public readonly requestId?: string;
  public readonly details?: unknown;

  constructor(message: string, code: number, options?: { requestId?: string; details?: unknown }) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.requestId = options?.requestId;
    this.details = options?.details;
  }

  /** 是否为认证错误 */
  get isAuthError(): boolean {
    return this.code === 401 || this.code === 2001;
  }

  /** 是否为权限错误 */
  get isForbidden(): boolean {
    return this.code === 403 || this.code === 2003;
  }

  /** 是否为未找到 */
  get isNotFound(): boolean {
    return this.code === 404 || this.code === 3001;
  }

  /** 是否为版本冲突 */
  get isConflict(): boolean {
    return this.code === 409;
  }
}

// ==================== BaseService 类 ====================

/**
 * BaseService - RESTful API 服务基类
 *
 * @template T - 实体类型
 * @template CreateDTO - 创建 DTO 类型
 * @template UpdateDTO - 更新 DTO 类型
 */
export abstract class BaseService<T, CreateDTO = Partial<T>, UpdateDTO = Partial<T>> {
  /** API 基础路径 */
  protected abstract readonly basePath: string;

  /**
   * 获取列表
   * GET /basePath
   */
  async list(params?: ListParams): Promise<PaginatedResponse<T>> {
    return httpClient.get<PaginatedResponse<T>>(this.basePath, params);
  }

  /**
   * 根据 ID 获取单个实体
   * GET /basePath/:id
   */
  async getById(id: number): Promise<T> {
    return httpClient.get<T>(`${this.basePath}/${id}`);
  }

  /**
   * 创建实体
   * POST /basePath
   */
  async create(data: CreateDTO): Promise<T> {
    return httpClient.post<T>(this.basePath, data);
  }

  /**
   * 更新实体
   * PUT /basePath/:id
   */
  async update(id: number, data: UpdateDTO): Promise<T> {
    return httpClient.put<T>(`${this.basePath}/${id}`, data);
  }

  /**
   * 部分更新实体
   * PATCH /basePath/:id
   */
  async patch(id: number, data: Partial<UpdateDTO>): Promise<T> {
    return httpClient.patch<T>(`${this.basePath}/${id}`, data);
  }

  /**
   * 删除实体
   * DELETE /basePath/:id
   */
  async delete(id: number): Promise<void> {
    return httpClient.delete(`${this.basePath}/${id}`);
  }

  /**
   * 批量删除
   * DELETE /basePath/batch
   */
  async batchDelete(ids: number[]): Promise<void> {
    return httpClient.request({
      method: 'DELETE',
      url: `${this.basePath}/batch`,
      data: { ids },
    });
  }

  /**
   * 检查实体是否存在
   */
  async exists(id: number): Promise<boolean> {
    try {
      await this.getById(id);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * 搜索实体
   * GET /basePath/search
   */
  async search(query: string, params?: ListParams): Promise<T[]> {
    return httpClient.get<T[]>(`${this.basePath}/search`, { q: query, ...params });
  }

  /**
   * 执行自定义 GET 请求
   */
  protected async get<R>(endpoint: string, params?: Record<string, unknown>): Promise<R> {
    return httpClient.get<R>(`${this.basePath}${endpoint}`, params);
  }

  /**
   * 执行自定义 POST 请求
   */
  protected async post<R>(endpoint: string, data?: unknown): Promise<R> {
    return httpClient.post<R>(`${this.basePath}${endpoint}`, data);
  }

  /**
   * 执行自定义 PUT 请求
   */
  protected async put<R>(endpoint: string, data?: unknown): Promise<R> {
    return httpClient.put<R>(`${this.basePath}${endpoint}`, data);
  }

  /**
   * 执行自定义 DELETE 请求
   */
  protected async del<R>(endpoint: string, data?: unknown): Promise<R> {
    return httpClient.delete<R>(`${this.basePath}${endpoint}`, data);
  }

  /**
   * 执行自定义 PATCH 请求
   */
  protected async patchEndpoint<R>(endpoint: string, data?: unknown): Promise<R> {
    return httpClient.patch<R>(`${this.basePath}${endpoint}`, data);
  }
}

// ==================== 工具函数 ====================

/**
 * 构建查询字符串
 */
export function buildQueryString(params: Record<string, unknown>): string {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      qs.append(key, String(value));
    }
  });
  return qs.toString();
}

/**
 * 合并路径
 */
export function joinPath(...parts: string[]): string {
  return parts
    .map((part, index) => {
      if (index === 0) return part.replace(/\/+$/, '');
      if (index === parts.length - 1) return part.replace(/^\/+/, '');
      return part.replace(/^\/+|\/+$/g, '');
    })
    .join('/');
}

export default BaseService;
