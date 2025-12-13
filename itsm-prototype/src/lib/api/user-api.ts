import { httpClient } from './http-client';

// 用户相关的接口定义
export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  active: boolean;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  password: string;
  tenant_id: number;
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
  name?: string;
  department?: string;
  phone?: string;
}

export interface ListUsersParams {
  page?: number;
  page_size?: number;
  tenant_id?: number;
  status?: string;
  department?: string;
  search?: string;
}

export interface PagedUsersResponse {
  users: User[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_page: number;
  };
}

export interface UserStatsResponse {
  total: number;
  active: number;
  inactive: number;
}

export interface BatchUpdateUsersRequest {
  user_ids: number[];
  action: 'activate' | 'deactivate' | 'department';
  department?: string;
}

export interface SearchUsersParams {
  keyword: string;
  tenant_id?: number;
  limit?: number;
}

export class UserApi {
  private static baseURL = '/api/v1/users';

  // 获取用户列表
  static async getUsers(params: ListUsersParams = {}): Promise<PagedUsersResponse> {
    return httpClient.get<PagedUsersResponse>(this.baseURL, params);
  }

  // 创建用户
  static async createUser(userData: CreateUserRequest): Promise<User> {
    return httpClient.post<User>(this.baseURL, userData);
  }

  // 获取用户详情
  static async getUserById(id: number): Promise<User> {
    return httpClient.get<User>(`${this.baseURL}/${id}`);
  }

  // 更新用户信息
  static async updateUser(id: number, userData: UpdateUserRequest): Promise<User> {
    return httpClient.put<User>(`${this.baseURL}/${id}`, userData);
  }

  // 删除用户（软删除）
  static async deleteUser(id: number): Promise<void> {
    await httpClient.delete(`${this.baseURL}/${id}`);
  }

  // 更改用户状态
  static async changeUserStatus(id: number, active: boolean): Promise<void> {
    await httpClient.put(`${this.baseURL}/${id}/status`, { active });
  }

  // 重置用户密码
  static async resetPassword(id: number, newPassword: string): Promise<void> {
    await httpClient.put(`${this.baseURL}/${id}/reset-password`, { new_password: newPassword });
  }

  // 获取用户统计
  static async getUserStats(tenantId?: number): Promise<UserStatsResponse> {
    const params = tenantId ? { tenant_id: tenantId } : {};
    return httpClient.get<UserStatsResponse>(`${this.baseURL}/stats`, params);
  }

  // 批量更新用户
  static async batchUpdateUsers(request: BatchUpdateUsersRequest): Promise<void> {
    await httpClient.put(`${this.baseURL}/batch`, request);
  }

  // 搜索用户
  static async searchUsers(params: SearchUsersParams): Promise<User[]> {
    return httpClient.get<User[]>(`${this.baseURL}/search`, params);
  }

  // 导出用户
  static async exportUsers(filters?: Record<string, unknown>): Promise<Blob> {
    const blob = await httpClient.request<Blob>({
      method: 'GET',
      url: `${this.baseURL}/export`,
      params: filters,
      responseType: 'blob',
    });
    return blob;
  }

  // 导入用户
  static async importUsers(file: File): Promise<{
    success: User[];
    failed: Array<{ index: number; user: CreateUserRequest; error: string }>;
    total: number;
    processed: number;
  }> {
    const formData = new FormData();
    formData.append('file', file);
    
    return httpClient.post(`${this.baseURL}/import`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    });
  }
}