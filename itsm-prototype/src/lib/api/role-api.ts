import { httpClient } from './http-client';
import {
  Role,
  RoleListResponse,
  CreateRoleRequest,
  UpdateRoleRequest,
  GetRolesParams,
} from './api-config';

export class RoleAPI {
  // 获取角色列表
  static async getRoles(params?: GetRolesParams): Promise<RoleListResponse> {
    return httpClient.get<RoleListResponse>('/api/roles', params);
  }

  // 获取单个角色
  static async getRole(id: number): Promise<Role> {
    return httpClient.get<Role>(`/api/roles/${id}`);
  }

  // 创建角色
  static async createRole(data: CreateRoleRequest): Promise<Role> {
    return httpClient.post<Role>('/api/roles', data);
  }

  // 更新角色
  static async updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
    return httpClient.put<Role>(`/api/roles/${id}`, data);
  }

  // 删除角色
  static async deleteRole(id: number): Promise<void> {
    return httpClient.delete<void>(`/api/roles/${id}`);
  }

  // 获取所有权限列表
  static async getPermissions(): Promise<string[]> {
    const response = await httpClient.get<{ permissions: string[] }>('/api/permissions');
    return response.data.permissions;
  }
}