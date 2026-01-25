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
    return await httpClient.get<RoleListResponse>('/api/v1/roles', params);
  }

  // 获取单个角色
  static async getRole(id: number): Promise<Role> {
    return await httpClient.get<Role>(`/api/v1/roles/${id}`);
  }

  // 创建角色
  static async createRole(data: CreateRoleRequest): Promise<Role> {
    return await httpClient.post<Role>('/api/v1/roles', data);
  }

  // 更新角色
  static async updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
    return await httpClient.put<Role>(`/api/v1/roles/${id}`, data);
  }

  // 删除角色
  static async deleteRole(id: number): Promise<void> {
    await httpClient.delete<void>(`/api/v1/roles/${id}`);
  }

  // 获取所有权限列表
  static async getPermissions(): Promise<string[]> {
    const response = await httpClient.get<{ permissions: string[] }>('/api/v1/permissions');
    return response.permissions;
  }
}