/**
 * 标准变更模板 API
 */

import { httpClient } from './http-client';

// Standard Change 请求接口
export interface StandardChangeRequest {
  title: string;
  description?: string;
  implementation_plan: string;
  rollback_plan: string;
  justification?: string;
  category?: string;
  risk_level?: string;
  impact_scope?: string;
  expected_duration?: number;
  approval_required?: boolean;
  affected_cis?: string[];
  prerequisites?: string[];
  remarks?: string;
}

// Standard Change 响应接口
export interface StandardChange {
  id: number;
  title: string;
  description: string;
  implementationPlan: string;
  rollbackPlan: string;
  justification: string;
  category: string;
  riskLevel: string;
  impactScope: string;
  expectedDuration: number;
  approvalRequired: boolean;
  affectedCIs: string[];
  prerequisites: string[];
  remarks: string;
  createdBy: number;
  tenantId: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// 更新请求接口
export interface UpdateStandardChangeRequest {
  title?: string;
  description?: string;
  implementation_plan?: string;
  rollback_plan?: string;
  justification?: string;
  category?: string;
  risk_level?: string;
  impact_scope?: string;
  expected_duration?: number;
  approval_required?: boolean;
  affected_cis?: string[];
  prerequisites?: string[];
  remarks?: string;
  is_active?: boolean;
}

// 实例化请求
export interface InstantiateStandardChangeRequest {
  title?: string;
  planned_start_date?: string;
  planned_end_date?: string;
  affected_cis?: string[];
}

// Standard Change 列表响应
export interface StandardChangeListResponse {
  total: number;
  templates: StandardChange[];
  page: number;
  page_size: number;
}

// Standard Change API 类
export class StandardChangeApi {
  // 获取标准变更模板列表
  static async getTemplates(params?: {
    page?: number;
    page_size?: number;
    category?: string;
    search?: string;
    active_only?: boolean;
  }): Promise<StandardChangeListResponse> {
    return httpClient.get<StandardChangeListResponse>('/api/v1/standard-changes', params);
  }

  // 获取单个标准变更模板
  static async getTemplate(id: number): Promise<StandardChange> {
    return httpClient.get<StandardChange>(`/api/v1/standard-changes/${id}`);
  }

  // 创建标准变更模板
  static async createTemplate(data: StandardChangeRequest): Promise<StandardChange> {
    return httpClient.post<StandardChange>('/api/v1/standard-changes', data);
  }

  // 更新标准变更模板
  static async updateTemplate(id: number, data: UpdateStandardChangeRequest): Promise<StandardChange> {
    return httpClient.put<StandardChange>(`/api/v1/standard-changes/${id}`, data);
  }

  // 删除标准变更模板 (软删除)
  static async deleteTemplate(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/standard-changes/${id}`);
  }

  // 获取分类列表
  static async getCategories(): Promise<{ categories: string[] }> {
    return httpClient.get<{ categories: string[] }>('/api/v1/standard-changes/categories');
  }

  // 从模板实例化变更
  static async instantiate(id: number, data?: InstantiateStandardChangeRequest): Promise<{ change_id: number }> {
    return httpClient.post<{ change_id: number }>(`/api/v1/standard-changes/${id}/instantiate`, data || {});
  }
}
