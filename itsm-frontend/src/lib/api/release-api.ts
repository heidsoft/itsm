/**
 * 发布管理 API 服务
 */

import { httpClient } from './http-client';

// 发布状态类型
export type ReleaseStatus =
  | 'draft'
  | 'scheduled'
  | 'in-progress'
  | 'completed'
  | 'cancelled'
  | 'failed'
  | 'rolled_back';

// 发布类型
export type ReleaseType = 'major' | 'minor' | 'patch' | 'hotfix';

// 发布严重程度
export type ReleaseSeverity = 'low' | 'medium' | 'high' | 'critical';

// 目标环境
export type ReleaseEnvironment = 'dev' | 'staging' | 'production';

// 发布请求接口
export interface ReleaseRequest {
  release_number: string;
  title: string;
  description?: string;
  type?: ReleaseType;
  environment?: ReleaseEnvironment;
  severity?: ReleaseSeverity;
  change_id?: number;
  owner_id?: number;
  planned_release_date?: string;
  planned_start_date?: string;
  planned_end_date?: string;
  release_notes?: string;
  rollback_procedure?: string;
  validation_criteria?: string;
  affected_systems?: string[];
  affected_components?: string[];
  deployment_steps?: string[];
  tags?: string[];
  is_emergency?: boolean;
  requires_approval?: boolean;
}

// 发布响应接口
export interface Release {
  id: number;
  release_number: string;
  title: string;
  description?: string;
  type: ReleaseType;
  status: ReleaseStatus;
  severity: ReleaseSeverity;
  environment: ReleaseEnvironment;
  change_id?: number;
  owner_id?: number;
  owner_name?: string;
  created_by: number;
  created_by_name: string;
  tenant_id: number;
  planned_release_date?: string;
  actual_release_date?: string;
  planned_start_date?: string;
  planned_end_date?: string;
  release_notes?: string;
  rollback_procedure?: string;
  validation_criteria?: string;
  affected_systems?: string[];
  affected_components?: string[];
  deployment_steps?: string[];
  tags?: string[];
  is_emergency: boolean;
  requires_approval: boolean;
  created_at: string;
  updated_at: string;
}

// 发布列表响应
export interface ReleaseListResponse {
  total: number;
  releases: Release[];
}

// 发布统计响应
export interface ReleaseStatsResponse {
  total: number;
  draft: number;
  scheduled: number;
  in_progress: number;
  completed: number;
  cancelled: number;
  failed: number;
  rolled_back: number;
}

// 发布状态更新请求
export interface ReleaseStatusUpdateRequest {
  status: ReleaseStatus;
  comment?: string;
}

// 发布API类
export class ReleaseApi {
  // 获取发布列表
  static async getReleases(params?: {
    page?: number;
    pageSize?: number;
    status?: ReleaseStatus;
    type?: ReleaseType;
  }): Promise<ReleaseListResponse> {
    return httpClient.get<ReleaseListResponse>('/api/v1/releases', params);
  }

  // 获取单个发布
  static async getRelease(id: number): Promise<Release> {
    return httpClient.get<Release>(`/api/v1/releases/${id}`);
  }

  // 创建发布
  static async createRelease(data: ReleaseRequest): Promise<Release> {
    return httpClient.post<Release>('/api/v1/releases', data);
  }

  // 更新发布
  static async updateRelease(id: number, data: Partial<ReleaseRequest>): Promise<Release> {
    return httpClient.put<Release>(`/api/v1/releases/${id}`, data);
  }

  // 删除发布
  static async deleteRelease(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/releases/${id}`);
  }

  // 更新发布状态
  static async updateReleaseStatus(id: number, status: ReleaseStatus): Promise<Release> {
    return httpClient.put<Release>(`/api/v1/releases/${id}/status`, { status });
  }

  // 获取发布统计
  static async getReleaseStats(): Promise<ReleaseStatsResponse> {
    return httpClient.get<ReleaseStatsResponse>('/api/v1/releases/stats');
  }
}
