import { httpClient } from '@/lib/api/http-client';

// 工单分类接口
export interface TicketCategory {
  id: number;
  name: string;
  description: string;
  code: string;
  parent_id: number;
  level: number;
  sort_order: number;
  is_active: boolean;
  tenant_id: number;
  created_at: string;
  updated_at: string;
  children?: TicketCategory[];
}

// 创建分类请求
export interface CreateCategoryRequest {
  name: string;
  description: string;
  code: string;
  parent_id: number;
  sort_order: number;
  is_active: boolean;
}

// 更新分类请求
export interface UpdateCategoryRequest {
  name?: string;
  description?: string;
  code?: string;
  parent_id?: number;
  sort_order?: number;
  is_active?: boolean;
}

// 分类列表请求参数
export interface ListCategoriesParams {
  page?: number;
  page_size?: number;
  parent_id?: number;
  level?: number;
  is_active?: boolean;
}

// 分类列表响应
export interface ListCategoriesResponse {
  categories: TicketCategory[];
  total: number;
}

// 分类树项目
export interface CategoryTreeItem {
  id: number;
  name: string;
  description: string;
  code: string;
  level: number;
  sort_order: number;
  sortOrder?: number;
  is_active: boolean;
  isActive?: boolean;
  children: CategoryTreeItem[];
  parent_id?: number;
  parentId?: number;
  tenant_id?: number;
  tenantId?: number;
  created_at?: string;
  createdAt?: string;
  updated_at?: string;
  updatedAt?: string;
  created_by?: string;
  createdBy?: string;
  updated_by?: string;
  updatedBy?: string;
}

// 移动分类请求
export interface MoveCategoryRequest {
  new_parent_id?: number;
  new_sort_order?: number;
}

// 批量更新分类请求
export interface BatchUpdateCategoriesRequest {
  id: number;
  parent_id?: number;
  sort_order: number;
  level: number;
}

class TicketCategoryService {
  private readonly baseUrl = '/api/v1/ticket-categories';

  // 获取分类列表
  async listCategories(params: ListCategoriesParams = {}): Promise<ListCategoriesResponse> {
    return httpClient.get<ListCategoriesResponse>(this.baseUrl, params);
  }

  // 获取分类树形结构
  async getCategoryTree(): Promise<CategoryTreeItem[]> {
    return httpClient.get<CategoryTreeItem[]>(`${this.baseUrl}/tree`);
  }

  // 获取分类详情
  async getCategory(id: number): Promise<TicketCategory> {
    return httpClient.get<TicketCategory>(`${this.baseUrl}/${id}`);
  }

  // 创建分类
  async createCategory(data: CreateCategoryRequest): Promise<TicketCategory> {
    return httpClient.post<TicketCategory>(this.baseUrl, data);
  }

  // 更新分类
  async updateCategory(id: number, data: UpdateCategoryRequest): Promise<TicketCategory> {
    return httpClient.put<TicketCategory>(`${this.baseUrl}/${id}`, data);
  }

  // 删除分类
  async deleteCategory(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }

  // 移动分类位置
  async moveCategory(id: number, data: MoveCategoryRequest): Promise<void> {
    return httpClient.put(`${this.baseUrl}/${id}/move`, data);
  }

  // 批量更新分类
  async batchUpdateCategories(data: BatchUpdateCategoriesRequest[]): Promise<void> {
    return httpClient.put(`${this.baseUrl}/batch-update`, data);
  }
}

export const ticketCategoryService = new TicketCategoryService();