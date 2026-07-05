/**
 * 工单分类 API 服务
 */

import { httpClient } from './http-client';

// 工单分类接口
export interface TicketCategory {
  id: number;
  name: string;
  description: string;
  parentId: number | null;
  level: number;
  path: string;
  sortOrder: number;
  isActive: boolean;
  isDefault: boolean;
  color: string;
  icon: string;
  ticketCount?: number;
  children?: TicketCategory[];
  createdAt: string;
  updatedAt: string;
}

// 创建分类请求
export interface CreateCategoryRequest {
  name: string;
  description?: string;
  parentId?: number;
  sortOrder?: number;
  isActive?: boolean;
  isDefault?: boolean;
  color?: string;
  icon?: string;
}

// 更新分类请求
export interface UpdateCategoryRequest {
  name?: string;
  description?: string;
  parentId?: number;
  sortOrder?: number;
  isActive?: boolean;
  isDefault?: boolean;
  color?: string;
  icon?: string;
}

export class TicketCategoryApi {
  // 获取分类列表 - 支持两种后端响应格式
  static async getCategories(params?: {
    page?: number;
    pageSize?: number;
    parentId?: number;
    isActive?: boolean;
    keyword?: string;
  }): Promise<{
    categories?: TicketCategory[];
    items?: TicketCategory[];
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
  static async previewImport(formData: FormData): Promise<
    {
      name: string;
      code: string;
      description: string;
      parentCode: string;
      sortOrder: number;
      isActive: boolean;
    }[]
  > {
    return httpClient.post('/api/v1/ticket-categories/import/preview', formData);
  }

  // 执行导入
  static async executeImport(formData: FormData): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/ticket-categories/import', formData);
  }
}

export default TicketCategoryApi;
