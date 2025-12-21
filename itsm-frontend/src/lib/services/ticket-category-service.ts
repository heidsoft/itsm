import { API_BASE_URL } from '@/app/lib/api-config';
import { getAccessToken } from '@/lib/auth/token-storage';

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
  is_active: boolean;
  children: CategoryTreeItem[];
  created_at?: string;
  updated_at?: string;
  created_by?: string;
  updated_by?: string;
}

// 移动分类请求
export interface MoveCategoryRequest {
  new_parent_id?: number;
  new_sort_order?: number;
}

class TicketCategoryService {
  private baseUrl = `${API_BASE_URL}/api/v1/ticket-categories`;

  // 获取分类列表
  async listCategories(params: ListCategoriesParams = {}): Promise<ListCategoriesResponse> {
    const queryParams = new URLSearchParams();
    
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.page_size) queryParams.append('page_size', params.page_size.toString());
    if (params.parent_id !== undefined) queryParams.append('parent_id', params.parent_id.toString());
    if (params.level) queryParams.append('level', params.level.toString());
    if (params.is_active !== undefined) queryParams.append('active', params.is_active.toString());

    const response = await fetch(`${this.baseUrl}?${queryParams}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      throw new Error(`获取分类列表失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 获取分类树形结构
  async getCategoryTree(): Promise<CategoryTreeItem[]> {
    const response = await fetch(`${this.baseUrl}/tree`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      throw new Error(`获取分类树失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 获取分类详情
  async getCategory(id: number): Promise<TicketCategory> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      throw new Error(`获取分类详情失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 创建分类
  async createCategory(data: CreateCategoryRequest): Promise<TicketCategory> {
    const response = await fetch(this.baseUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || `创建分类失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 更新分类
  async updateCategory(id: number, data: UpdateCategoryRequest): Promise<TicketCategory> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || `更新分类失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 删除分类
  async deleteCategory(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || `删除分类失败: ${response.statusText}`);
    }
  }

  // 移动分类位置
  async moveCategory(id: number, data: MoveCategoryRequest): Promise<void> {
    const response = await fetch(`${this.baseUrl}/${id}/move`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getAuthToken()}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || `移动分类失败: ${response.statusText}`);
    }
  }

  // 获取认证token
  private getAuthToken(): string {
    return getAccessToken() || '';
  }
}

export const ticketCategoryService = new TicketCategoryService();
