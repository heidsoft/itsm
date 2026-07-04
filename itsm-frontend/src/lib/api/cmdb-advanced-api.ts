/**
 * CMDB 高级 API 服务
 * 补充缺失的后端接口对接
 * 后端API基路径: /api/v1/cmdb
 */

import { httpClient } from './http-client';

// ==================== 类型定义 ====================

/** CI标签 */
export interface CITag {
  id: number;
  key: string;
  value: string;
  color?: string;
  description?: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

/** CI标签列表响应 */
export interface CITagListResponse {
  items: CITag[];
  total: number;
  page: number;
  size: number;
}

/** 创建标签请求 */
export interface CreateTagRequest {
  key: string;
  value: string;
  color?: string;
  description?: string;
}

/** 更新标签请求 */
export interface UpdateTagRequest {
  key?: string;
  value?: string;
  color?: string;
  description?: string;
}

/** 保存的视图 */
export interface CISavedView {
  id: number;
  name: string;
  description?: string;
  filters: CISearchFilter;
  sortBy?: string;
  sortOrder?: string;
  isPublic: boolean;
  creatorId: number;
  creatorName?: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

/** CI搜索过滤器 */
export interface CISearchFilter {
  keyword?: string;
  ci_type_id?: number;
  status?: string;
  environment?: string;
  criticality?: string;
  asset_tag?: string;
  serial_number?: string;
  vendor?: string;
  location?: string;
  assigned_to?: string;
  owned_by?: string;
  tag_ids?: number[];
  attributes?: Record<string, unknown>;
  date_from?: string;
  date_to?: string;
  cloud_provider?: string;
  cloud_region?: string;
  cloud_resource_id?: string;
}

/** 导入任务 */
export interface ImportTask {
  task_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  total_count: number;
  success_count: number;
  failed_count: number;
  errors?: ImportError[];
  created_at: string;
  completed_at?: string;
}

/** 导入错误 */
export interface ImportError {
  row_number: number;
  field_name: string;
  message: string;
}

/** 导出任务 */
export interface ExportTask {
  task_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  file_url?: string;
  total_count: number;
  file_size?: number;
  created_at: string;
  completed_at?: string;
  expires_at: string;
}

/** 导入请求 */
export interface CreateImportRequest {
  file_url: string;
  file_type: 'xlsx' | 'csv';
  update_mode: 'skip' | 'overwrite' | 'merge';
  sheet_name?: string;
}

/** 导出请求 */
export interface CreateExportRequest {
  filters?: CISearchFilter;
  export_type: 'xlsx' | 'csv';
  export_fields?: string[];
}

/** CI属性定义 */
export interface CIAttributeDefinition {
  id: number;
  name: string;
  display_name: string;
  data_type: string;
  is_required: boolean;
  is_unique: boolean;
  default_value?: string;
  options?: AttributeOption[];
  validation_rule?: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

/** 属性选项 */
export interface AttributeOption {
  label: string;
  value: string;
}

/** 生命周期状态 */
export type LifecycleStatus = 'draft' | 'online' | 'maintenance' | 'offline' | 'scrapped';

/** 生命周期历史记录 */
export interface LifecycleHistoryItem {
  id: number;
  ci_id: number;
  from_status: string;
  to_status: string;
  changed_by: string;
  changed_at: string;
  reason?: string;
}

/** 版本信息 */
export interface CIVersion {
  version: number;
  ci_id: number;
  name: string;
  status: string;
  attributes: Record<string, unknown>;
  created_by: string;
  created_at: string;
}

// ==================== API 服务 ====================

const CMDB_BASE = '/api/v1/cmdb';

/**
 * CMDB 高级 API
 * 补充缺失的API接口
 */
export class CMDBAdvancedApi {
  // ==================== 标签管理 ====================

  /** 获取标签列表 */
  static async getTags(params?: {
    page?: number;
    page_size?: number;
    keyword?: string;
  }): Promise<CITagListResponse> {
    return httpClient.get(`${CMDB_BASE}/tags`, params);
  }

  /** 获取标签详情 */
  static async getTag(id: number): Promise<CITag> {
    return httpClient.get(`${CMDB_BASE}/tags/${id}`);
  }

  /** 创建标签 */
  static async createTag(data: CreateTagRequest): Promise<CITag> {
    return httpClient.post(`${CMDB_BASE}/tags`, data);
  }

  /** 更新标签 */
  static async updateTag(id: number, data: UpdateTagRequest): Promise<CITag> {
    return httpClient.put(`${CMDB_BASE}/tags/${id}`, data);
  }

  /** 删除标签 */
  static async deleteTag(id: number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/tags/${id}`);
  }

  // ==================== 保存视图 ====================

  /** 获取视图列表 */
  static async getViews(params?: {
    page?: number;
    page_size?: number;
  }): Promise<{
    items: CISavedView[];
    total: number;
    page: number;
    size: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/views`, params);
  }

  /** 获取视图详情 */
  static async getView(id: number): Promise<CISavedView> {
    return httpClient.get(`${CMDB_BASE}/views/${id}`);
  }

  /** 创建视图 */
  static async createView(data: {
    name: string;
    description?: string;
    filters: CISearchFilter;
    sortBy?: string;
    sortOrder?: string;
    isPublic?: boolean;
  }): Promise<CISavedView> {
    return httpClient.post(`${CMDB_BASE}/views`, data);
  }

  /** 更新视图 */
  static async updateView(id: number, data: Partial<{
    name: string;
    description: string;
    filters: CISearchFilter;
    sortBy: string;
    sortOrder: string;
    isPublic: boolean;
  }>): Promise<CISavedView> {
    return httpClient.put(`${CMDB_BASE}/views/${id}`, data);
  }

  /** 删除视图 */
  static async deleteView(id: number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/views/${id}`);
  }

  // ==================== 导入导出 ====================

  /** 获取导入任务列表 */
  static async getImportTasks(params?: {
    page?: number;
    page_size?: number;
  }): Promise<{
    items: ImportTask[];
    total: number;
    page: number;
    size: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/import`, params);
  }

  /** 获取导入任务状态 */
  static async getImportTaskStatus(taskId: string): Promise<ImportTask> {
    return httpClient.get(`${CMDB_BASE}/import/${taskId}`);
  }

  /** 创建导入任务 */
  static async createImportTask(data: CreateImportRequest): Promise<ImportTask> {
    return httpClient.post(`${CMDB_BASE}/import`, data);
  }

  /** 获取导出任务列表 */
  static async getExportTasks(params?: {
    page?: number;
    page_size?: number;
  }): Promise<{
    items: ExportTask[];
    total: number;
    page: number;
    size: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/export`, params);
  }

  /** 获取导出任务状态 */
  static async getExportTaskStatus(taskId: string): Promise<ExportTask> {
    return httpClient.get(`${CMDB_BASE}/export/${taskId}`);
  }

  /** 创建导出任务 */
  static async createExportTask(data: CreateExportRequest): Promise<ExportTask> {
    return httpClient.post(`${CMDB_BASE}/export`, data);
  }

  // ==================== 批量操作 ====================

  /** 批量创建CI */
  static async batchCreateCI(items: Array<{
    name: string;
    ciTypeId: number;
    status: string;
    [key: string]: unknown;
  }>): Promise<{
    items: Array<{ id: number; name: string; success: boolean; error?: string }>;
    total: number;
    success_count: number;
    failed_count: number;
  }> {
    return httpClient.post(`${CMDB_BASE}/cis/batch`, { items });
  }

  /** 批量更新CI */
  static async batchUpdateCI(items: Array<{
    id: number;
    [key: string]: unknown;
  }>): Promise<{
    items: Array<{ id: number; success: boolean; error?: string }>;
    total: number;
    success_count: number;
    failed_count: number;
  }> {
    return httpClient.put(`${CMDB_BASE}/cis/batch`, { items });
  }

  /** 批量删除CI */
  static async batchDeleteCI(ids: number[]): Promise<{
    deleted_count: number;
    failed_ids: number[];
  }> {
    return httpClient.delete(`${CMDB_BASE}/cis/batch`, { ids });
  }

  /** 批量更新生命周期状态 */
  static async batchUpdateLifecycle(
    ids: number[],
    status: LifecycleStatus,
    reason?: string
  ): Promise<{
    updated_count: number;
    failed_ids: number[];
  }> {
    return httpClient.put(`${CMDB_BASE}/cis/batch/lifecycle`, {
      ids,
      status,
      reason,
    });
  }

  // ==================== 生命周期管理 ====================

  /** 获取CI生命周期历史 */
  static async getLifecycleHistory(ciId: number): Promise<{
    items: LifecycleHistoryItem[];
    total: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/cis/${ciId}/lifecycle/history`);
  }

  /** 更新CI生命周期状态 */
  static async updateLifecycleStatus(
    ciId: number,
    status: LifecycleStatus,
    reason?: string
  ): Promise<{
    ci_id: number;
    status: LifecycleStatus;
    changed_at: string;
  }> {
    return httpClient.put(`${CMDB_BASE}/cis/${ciId}/lifecycle`, {
      status,
      reason,
    });
  }

  // ==================== 版本管理 ====================

  /** 回滚CI版本 */
  static async revertCIVersion(
    ciId: number,
    version: number,
    reason?: string
  ): Promise<{
    id: number;
    name: string;
    version: number;
    reverted_at: string;
  }> {
    return httpClient.post(`${CMDB_BASE}/cis/${ciId}/revert`, {
      version,
      reason,
    });
  }

  // ==================== 标签关联 ====================

  /** 给CI添加标签 */
  static async addTagsToCI(ciId: number, tagIds: number[]): Promise<{
    ci_id: number;
    tag_ids: number[];
  }> {
    return httpClient.post(`${CMDB_BASE}/cis/${ciId}/tags`, { tagIds });
  }

  /** 从CI移除标签 */
  static async removeTagsFromCI(ciId: number, tagIds: number[]): Promise<{
    ci_id: number;
    removed_tag_ids: number[];
  }> {
    return httpClient.delete(`${CMDB_BASE}/cis/${ciId}/tags`, { tagIds });
  }

  // ==================== CI属性定义 ====================

  /** 获取CI类型的属性定义 */
  static async getCITypeAttributes(ciTypeId: number): Promise<{
    items: CIAttributeDefinition[];
    total: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/ci-types/${ciTypeId}/attributes`);
  }

  /** 获取属性定义详情 */
  static async getAttributeDefinition(id: number): Promise<CIAttributeDefinition> {
    return httpClient.get(`${CMDB_BASE}/attributes/${id}`);
  }

  /** 创建属性定义 */
  static async createAttributeDefinition(data: {
    ci_type_id: number;
    name: string;
    display_name: string;
    data_type: string;
    is_required?: boolean;
    is_unique?: boolean;
    default_value?: string;
    options?: AttributeOption[];
    validation_rule?: string;
    description?: string;
  }): Promise<CIAttributeDefinition> {
    return httpClient.post(`${CMDB_BASE}/attributes`, data);
  }

  /** 更新属性定义 */
  static async updateAttributeDefinition(
    id: number,
    data: Partial<{
      name: string;
      display_name: string;
      data_type: string;
      is_required: boolean;
      is_unique: boolean;
      default_value: string;
      options: AttributeOption[];
      validation_rule: string;
      description: string;
    }>
  ): Promise<CIAttributeDefinition> {
    return httpClient.put(`${CMDB_BASE}/attributes/${id}`, data);
  }

  /** 删除属性定义 */
  static async deleteAttributeDefinition(id: number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/attributes/${id}`);
  }

  // ==================== CI关系完整版 ====================

  /** 获取CI关系列表 */
  static async getRelationships(params?: {
    page?: number;
    page_size?: number;
    source_ci_id?: number;
    target_ci_id?: number;
    relationship_type?: string;
  }): Promise<{
    items: Array<{
      id: number;
      source_ci_id: number;
      source_ci_name: string;
      target_ci_id: number;
      target_ci_name: string;
      relationship_type: string;
      description: string;
      created_at: string;
    }>;
    total: number;
  }> {
    return httpClient.get(`${CMDB_BASE}/relationships`, params);
  }

  /** 获取CI关系详情 */
  static async getRelationship(id: number): Promise<{
    id: number;
    source_ci_id: number;
    target_ci_id: number;
    relationship_type: string;
    description: string;
    created_at: string;
  }> {
    return httpClient.get(`${CMDB_BASE}/relationships/${id}`);
  }

  /** 更新CI关系 */
  static async updateRelationship(
    id: number,
    data: {
      relationship_type?: string;
      description?: string;
    }
  ): Promise<{
    id: number;
    success: boolean;
  }> {
    return httpClient.put(`${CMDB_BASE}/relationships/${id}`, data);
  }

  /** 删除CI关系 */
  static async deleteRelationship(id: number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/relationships/${id}`);
  }
}

export default CMDBAdvancedApi;
export const CMDB_ADVANCED_API = CMDBAdvancedApi;
