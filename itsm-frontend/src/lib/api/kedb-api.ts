/**
 * 已知错误数据库 (KEDB) API
 */

import { httpClient } from './http-client';

// KEDB 创建请求
export interface KEDBCreateRequest {
  title: string;
  description?: string;
  symptoms?: string;
  rootCause?: string;
  workaround?: string;
  resolution?: string;
  category?: string;
  severity?: string;
  affectedProducts?: string[];
  affectedCis?: string[];
  keywords?: string[];
  problemId?: number;
}

// KEDB 更新请求
export interface KEDBUpdateRequest {
  title?: string;
  description?: string;
  symptoms?: string;
  rootCause?: string;
  workaround?: string;
  resolution?: string;
  category?: string;
  severity?: string;
  status?: string;
  affectedProducts?: string[];
  affectedCis?: string[];
  keywords?: string[];
  isKnownError?: boolean;
}

// KEDB 响应
export interface KEDBResponse {
  id: number;
  title: string;
  description: string;
  symptoms: string;
  rootCause: string;
  workaround: string;
  resolution: string;
  status: string;
  category: string;
  severity: string;
  affectedProducts: string[];
  affectedCis: string[];
  keywords: string[];
  occurrenceCount: number;
  problemId?: number;
  createdBy: number;
  tenantId: number;
  isKnownError: boolean;
  createdAt: string;
  updatedAt: string;
}

// KEDB 列表响应（后端 common.SuccessWithList 标准信封）
export interface KEDBListResponse {
  items: KEDBResponse[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// KEDB 统计响应：聚合计数对象，不带分页字段
export interface KEDBStatsResponse {
  total: number;
  active: number;
  resolved: number;
  deprecated: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
}

// KEDB API 类
export class KEDBApi {
  // 获取已知错误列表
  static async getKnownErrors(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
    category?: string;
    severity?: string;
    keyword?: string;
  }): Promise<KEDBListResponse> {
    return httpClient.get<KEDBListResponse>('/api/v1/known-errors', params);
  }

  // 获取单个已知错误
  static async getKnownError(id: number): Promise<KEDBResponse> {
    return httpClient.get<KEDBResponse>(`/api/v1/known-errors/${id}`);
  }

  // 创建已知错误
  static async createKnownError(data: KEDBCreateRequest): Promise<KEDBResponse> {
    return httpClient.post<KEDBResponse>('/api/v1/known-errors', data);
  }

  // 更新已知错误
  static async updateKnownError(id: number, data: KEDBUpdateRequest): Promise<KEDBResponse> {
    return httpClient.put<KEDBResponse>(`/api/v1/known-errors/${id}`, data);
  }

  // 删除已知错误
  static async deleteKnownError(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/known-errors/${id}`);
  }

  // 获取统计信息
  static async getStats(): Promise<KEDBStatsResponse> {
    return httpClient.get<KEDBStatsResponse>('/api/v1/known-errors/stats');
  }

  // 搜索已知错误（与列表同一标准信封）
  static async searchKnownErrors(query: string): Promise<KEDBListResponse> {
    return httpClient.get<KEDBListResponse>('/api/v1/known-errors/search', { q: query });
  }

  // 获取分类列表（全量，不分页）
  static async getCategories(): Promise<{ items: string[]; total: number }> {
    return httpClient.get<{ items: string[]; total: number }>('/api/v1/known-errors/categories');
  }

  // 晋升为正式已知错误
  static async promoteToKnownError(id: number): Promise<{ message: string }> {
    return httpClient.post<{ message: string }>(`/api/v1/known-errors/${id}/promote`, {});
  }
}