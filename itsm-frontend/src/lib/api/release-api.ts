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
  releaseNumber: string;
  title: string;
  description?: string;
  type?: ReleaseType;
  environment?: ReleaseEnvironment;
  severity?: ReleaseSeverity;
  changeId?: number;
  ownerId?: number;
  plannedReleaseDate?: string;
  plannedStartDate?: string;
  plannedEndDate?: string;
  releaseNotes?: string;
  rollbackProcedure?: string;
  validationCriteria?: string;
  affectedSystems?: string[];
  affectedComponents?: string[];
  deploymentSteps?: string[];
  tags?: string[];
  isEmergency?: boolean;
  requiresApproval?: boolean;
}

// 发布响应接口
export interface Release {
  id: number;
  releaseNumber: string;
  title: string;
  description?: string;
  type: ReleaseType;
  status: ReleaseStatus;
  severity: ReleaseSeverity;
  environment: ReleaseEnvironment;
  changeId?: number;
  ownerId?: number;
  ownerName?: string;
  createdBy: number;
  createdByName: string;
  tenantId: number;
  plannedReleaseDate?: string;
  actualReleaseDate?: string;
  plannedStartDate?: string;
  plannedEndDate?: string;
  releaseNotes?: string;
  rollbackProcedure?: string;
  validationCriteria?: string;
  affectedSystems?: string[];
  affectedComponents?: string[];
  deploymentSteps?: string[];
  tags?: string[];
  isEmergency: boolean;
  requiresApproval: boolean;
  createdAt: string;
  updatedAt: string;
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
  inProgress: number;
  completed: number;
  cancelled: number;
  failed: number;
  rolledBack: number;
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
