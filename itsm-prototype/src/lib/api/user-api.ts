import { httpClient } from '../../app/lib/http-client';
import { 
  User, 
  CreateUserRequest, 
  UpdateUserRequest, 
  UserFilters, 
  UserListResponse,
  UserStats,
  UserActivity,
  UserSession,
  Permission,
  UserPreferences,
  UserImportRequest,
  UserExportRequest
} from '../../types/user';

/**
 * 用户API客户端
 * 提供用户管理相关的API调用方法
 */
export class UserAPI {
  /**
   * 获取用户列表
   * @param params 查询参数
   * @returns 用户列表响应
   */
  static async listUsers(params: UserFilters = {}): Promise<UserListResponse> {
    return httpClient.get<UserListResponse>('/api/v1/users', params as Record<string, unknown>);
  }

  /**
   * 获取单个用户详情
   * @param id 用户ID
   * @returns 用户详情响应
   */
  static async getUser(id: number): Promise<{ user: User }> {
    return httpClient.get<{ user: User }>(`/api/v1/users/${id}`);
  }

  /**
   * 创建新用户
   * @param data 用户创建数据
   * @returns 用户响应
   */
  static async createUser(data: CreateUserRequest): Promise<{ user: User }> {
    return httpClient.post<{ user: User }>('/api/v1/users', data);
  }

  /**
   * 更新用户信息
   * @param id 用户ID
   * @param data 用户更新数据
   * @returns 用户响应
   */
  static async updateUser(id: number, data: UpdateUserRequest): Promise<{ user: User }> {
    return httpClient.put<{ user: User }>(`/api/v1/users/${id}`, data);
  }

  /**
   * 删除用户
   * @param id 用户ID
   */
  static async deleteUser(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/users/${id}`);
  }

  /**
   * 批量删除用户
   * @param ids 用户ID数组
   */
  static async batchDeleteUsers(ids: number[]): Promise<void> {
    return httpClient.post('/api/v1/users/batch-delete', { ids });
  }

  /**
   * 启用用户
   * @param id 用户ID
   * @returns 用户响应
   */
  static async enableUser(id: number): Promise<{ user: User }> {
    return httpClient.post<{ user: User }>(`/api/v1/users/${id}/enable`);
  }

  /**
   * 禁用用户
   * @param id 用户ID
   * @param reason 禁用原因
   * @returns 用户响应
   */
  static async disableUser(id: number, reason?: string): Promise<{ user: User }> {
    return httpClient.post<{ user: User }>(`/api/v1/users/${id}/disable`, { reason });
  }

  /**
   * 重置用户密码
   * @param id 用户ID
   * @param newPassword 新密码
   * @returns 用户响应
   */
  static async resetPassword(id: number, newPassword: string): Promise<{ user: User }> {
    return httpClient.post<{ user: User }>(`/api/v1/users/${id}/reset-password`, { 
      new_password: newPassword 
    });
  }

  /**
   * 更新用户角色
   * @param id 用户ID
   * @param roleIds 角色ID数组
   * @returns 用户响应
   */
  static async updateUserRoles(id: number, roleIds: number[]): Promise<{ user: User }> {
    return httpClient.post<{ user: User }>(`/api/v1/users/${id}/roles`, { role_ids: roleIds });
  }

  /**
   * 获取用户权限
   * @param id 用户ID
   * @returns 用户权限列表
   */
  static async getUserPermissions(id: number): Promise<Permission[]> {
    return httpClient.get<Permission[]>(`/api/v1/users/${id}/permissions`);
  }

  /**
   * 获取用户活动记录
   * @param id 用户ID
   * @param params 查询参数
   * @returns 用户活动列表
   */
  static async getUserActivities(id: number, params: { 
    page?: number; 
    page_size?: number; 
    activity_type?: string;
    date_from?: string;
    date_to?: string;
  } = {}): Promise<{ activities: UserActivity[]; total: number }> {
    return httpClient.get<{ activities: UserActivity[]; total: number }>(
      `/api/v1/users/${id}/activities`, 
      params as Record<string, unknown>
    );
  }

  /**
   * 获取用户会话信息
   * @param id 用户ID
   * @returns 用户会话列表
   */
  static async getUserSessions(id: number): Promise<UserSession[]> {
    return httpClient.get<UserSession[]>(`/api/v1/users/${id}/sessions`);
  }

  /**
   * 终止用户会话
   * @param id 用户ID
   * @param sessionId 会话ID
   */
  static async terminateUserSession(id: number, sessionId: string): Promise<void> {
    return httpClient.delete(`/api/v1/users/${id}/sessions/${sessionId}`);
  }

  /**
   * 获取用户偏好设置
   * @param id 用户ID
   * @returns 用户偏好设置
   */
  static async getUserPreferences(id: number): Promise<UserPreferences> {
    return httpClient.get<UserPreferences>(`/api/v1/users/${id}/preferences`);
  }

  /**
   * 更新用户偏好设置
   * @param id 用户ID
   * @param preferences 偏好设置
   * @returns 用户偏好设置
   */
  static async updateUserPreferences(id: number, preferences: Partial<UserPreferences>): Promise<UserPreferences> {
    return httpClient.put<UserPreferences>(`/api/v1/users/${id}/preferences`, preferences);
  }

  /**
   * 获取用户统计信息
   * @returns 用户统计数据
   */
  static async getUserStats(): Promise<UserStats> {
    return httpClient.get<UserStats>('/api/v1/users/stats');
  }

  /**
   * 导入用户
   * @param data 导入数据
   * @returns 导入结果
   */
  static async importUsers(data: UserImportRequest): Promise<{
    success_count: number;
    error_count: number;
    errors: Array<{ row: number; message: string }>;
  }> {
    return httpClient.post('/api/v1/users/import', data);
  }

  /**
   * 导出用户
   * @param params 导出参数
   * @returns 导出文件Blob
   */
  static async exportUsers(params: UserExportRequest): Promise<Blob> {
    const response = await httpClient.get<ArrayBuffer>('/api/v1/users/export', params as unknown as Record<string, unknown>);
    return new Blob([response], { type: 'application/octet-stream' });
  }

  /**
   * 搜索用户
   * @param query 搜索关键词
   * @param limit 结果限制数量
   * @returns 用户列表
   */
  static async searchUsers(query: string, limit: number = 10): Promise<User[]> {
    return httpClient.get<User[]>('/api/v1/users/search', { q: query, limit });
  }

  /**
   * 验证用户名是否可用
   * @param username 用户名
   * @returns 是否可用
   */
  static async checkUsernameAvailability(username: string): Promise<{ available: boolean }> {
    return httpClient.get<{ available: boolean }>('/api/v1/users/check-username', { username });
  }

  /**
   * 验证邮箱是否可用
   * @param email 邮箱
   * @returns 是否可用
   */
  static async checkEmailAvailability(email: string): Promise<{ available: boolean }> {
    return httpClient.get<{ available: boolean }>('/api/v1/users/check-email', { email });
  }
}