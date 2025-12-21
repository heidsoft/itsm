import { httpClient } from '@/lib/api/http-client';
import { ApiResponse, PaginationResponse } from '@/app/lib/api-config';

/**
 * 基础API类，提供通用的CRUD操作
 */
export abstract class BaseApi {
  protected static baseUrl: string = '';

  /**
   * 获取列表数据
   */
  protected static async getList<T, P = Record<string, unknown>>(
    endpoint: string,
    params?: P
  ): Promise<PaginationResponse<T>> {
    try {
      const response = await httpClient.get<PaginationResponse<T>>(
        endpoint,
        (params as unknown as Record<string, unknown>) || undefined
      );
      return response;
    } catch (error) {
      console.error(`BaseApi.getList error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 获取单个资源
   */
  protected static async getById<T>(endpoint: string, id: string | number): Promise<T> {
    try {
      const response = await httpClient.get<T>(`${endpoint}/${id}`);
      return response;
    } catch (error) {
      console.error(`BaseApi.getById error for ${endpoint}/${id}:`, error);
      throw error;
    }
  }

  /**
   * 创建资源
   */
  protected static async create<T, D = Record<string, unknown>>(
    endpoint: string,
    data: D
  ): Promise<T> {
    try {
      const response = await httpClient.post<T>(endpoint, data);
      return response;
    } catch (error) {
      console.error(`BaseApi.create error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 更新资源
   */
  protected static async update<T, D = Record<string, unknown>>(
    endpoint: string,
    id: string | number,
    data: D
  ): Promise<T> {
    try {
      const response = await httpClient.put<T>(`${endpoint}/${id}`, data);
      return response;
    } catch (error) {
      console.error(`BaseApi.update error for ${endpoint}/${id}:`, error);
      throw error;
    }
  }

  /**
   * 部分更新资源
   */
  protected static async patch<T, D = Record<string, unknown>>(
    endpoint: string,
    id: string | number,
    data: D
  ): Promise<T> {
    try {
      const response = await httpClient.patch<T>(`${endpoint}/${id}`, data);
      return response;
    } catch (error) {
      console.error(`BaseApi.patch error for ${endpoint}/${id}:`, error);
      throw error;
    }
  }

  /**
   * 删除资源
   */
  protected static async delete<T = void>(endpoint: string, id: string | number): Promise<T> {
    try {
      const response = await httpClient.delete<T>(`${endpoint}/${id}`);
      return response;
    } catch (error) {
      console.error(`BaseApi.delete error for ${endpoint}/${id}:`, error);
      throw error;
    }
  }

  /**
   * 批量删除资源
   */
  protected static async batchDelete<T = void>(
    endpoint: string,
    ids: (string | number)[]
  ): Promise<T> {
    try {
      const response = await httpClient.delete<T>(`${endpoint}/batch`, { ids });
      return response;
    } catch (error) {
      console.error(`BaseApi.batchDelete error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 搜索资源
   */
  protected static async search<T, P = Record<string, unknown>>(
    endpoint: string,
    params: P
  ): Promise<PaginationResponse<T>> {
    try {
      const response = await httpClient.get<PaginationResponse<T>>(
        `${endpoint}/search`,
        params as unknown as Record<string, unknown>
      );
      return response;
    } catch (error) {
      console.error(`BaseApi.search error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 获取统计数据
   */
  protected static async getStats<T>(endpoint: string, params?: Record<string, unknown>): Promise<T> {
    try {
      const response = await httpClient.get<T>(`${endpoint}/stats`, params);
      return response;
    } catch (error) {
      console.error(`BaseApi.getStats error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 导出数据
   */
  protected static async export<P = Record<string, unknown>>(
    endpoint: string,
    params?: P,
    format: 'csv' | 'excel' | 'pdf' = 'csv'
  ): Promise<Blob> {
    try {
      const response = await fetch(`${httpClient.getBaseURL()}${endpoint}/export?format=${format}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${httpClient.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Export failed: ${response.statusText}`);
      }

      return await response.blob();
    } catch (error) {
      console.error(`BaseApi.export error for ${endpoint}:`, error);
      throw error;
    }
  }

  /**
   * 上传文件
   */
  protected static async upload<T>(
    endpoint: string,
    file: File,
    additionalData?: Record<string, unknown>
  ): Promise<T> {
    try {
      const formData = new FormData();
      formData.append('file', file);
      
      if (additionalData) {
        Object.entries(additionalData).forEach(([key, value]) => {
          formData.append(key, String(value));
        });
      }

      const response = await httpClient.post<T>(endpoint, formData);
      return response;
    } catch (error) {
      console.error(`BaseApi.upload error for ${endpoint}:`, error);
      throw error;
    }
  }
}

/**
 * API错误处理工具
 */
export class ApiError extends Error {
  public code: number;
  public originalError?: unknown;

  constructor(message: string, code: number = 5001, originalError?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.originalError = originalError;
  }

  static fromResponse(response: ApiResponse<unknown>): ApiError {
    return new ApiError(response.message, response.code);
  }

  static fromError(error: unknown): ApiError {
    if (error instanceof ApiError) {
      return error;
    }
    
    if (error instanceof Error) {
      return new ApiError(error.message, 5001, error);
    }
    
    return new ApiError('Unknown error occurred', 5001, error);
  }
}

/**
 * API响应处理工具
 */
export class ApiResponseHandler {
  /**
   * 检查响应是否成功
   */
  static isSuccess(response: ApiResponse<unknown>): boolean {
    return response.code === 0;
  }

  /**
   * 提取响应数据
   */
  static extractData<T>(response: ApiResponse<T>): T {
    if (!this.isSuccess(response)) {
      throw ApiError.fromResponse(response);
    }
    return response.data;
  }

  /**
   * 处理分页响应
   */
  static handlePaginationResponse<T>(
    response: ApiResponse<PaginationResponse<T>>
  ): PaginationResponse<T> {
    return this.extractData(response);
  }
}