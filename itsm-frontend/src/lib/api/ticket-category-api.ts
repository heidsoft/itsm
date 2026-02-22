/**
 * 工单分类 API 服务
 */

import { httpClient } from './http-client';

// 工单分类接口
export interface TicketCategory {
  id: number;
  name: string;
  description: string;
  parent_id: number | null;
  level: number;
  path: string;
  sort_order: number;
  is_active: boolean;
  is_default: boolean;
  color: string;
  icon: string;
  ticket_count?: number;
  children?: TicketCategory[];
  created_at: string;
  updated_at: string;
}

// 创建分类请求
export interface CreateCategoryRequest {
  name: string;
  description?: string;
  parent_id?: number;
  sort_order?: number;
  is_active?: boolean;
  is_default?: boolean;
  color?: string;
  icon?: string;
}

// 更新分类请求
export interface UpdateCategoryRequest {
  name?: string;
  description?: string;
  parent_id?: number;
  sort_order?: number;
  is_active?: boolean;
  is_default?: boolean;
  color?: string;
  icon?: string;
}

export class TicketCategoryApi {
  // 获取分类列表
  static async getCategories(params?: {
    page?: number;
    page_size?: number;
    parent_id?: number;
    is_active?: boolean;
    keyword?: string;
  }): Promise<{
    items: TicketCategory[];
    total: number;
  }> {
    return httpClient.get('/api/v1/ticket-categories', params);
  }

  // 获取分类树形结构
  static async getCategoryTree(): Promise<TicketCategory[]> {
    return httpClient.get('/api/v1/ticket-categories/tree');
  }

  // 获取单个分类
  static async getCategory(id: number): Promise<TicketCategory> {
    return httpClient.get(`/api/v1/ticket-categories/${id}`);
  }

  // 创建分类
  static async createCategory(data: CreateCategoryRequest): Promise<TicketCategory> {
    return httpClient.post('/api/v1/ticket-categories', data);
  }

  // 更新分类
  static async updateCategory(id: number, data: UpdateCategoryRequest): Promise<TicketCategory> {
    return httpClient.put(`/api/v1/ticket-categories/${id}`, data);
  }

  // 删除分类
  static async deleteCategory(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/ticket-categories/${id}`);
  }

  // 预览导入数据
  static async previewImport(formData: FormData): Promise<{
    name: string;
    code: string;
    description: string;
    parent_code: string;
    sort_order: number;
    is_active: boolean;
  }[]> {
    return httpClient.post('/api/v1/ticket-categories/import/preview', formData);
  }

  // 执行导入
  static async executeImport(formData: FormData): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/ticket-categories/import', formData);
  }
}

export default TicketCategoryApi;
