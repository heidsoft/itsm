import { httpClient } from './http-client';
import {
  Role,
  RoleListResponse,
  CreateRoleRequest,
  UpdateRoleRequest,
  GetRolesParams,
  PermissionCatalogItem,
} from './api-config';

export class RoleAPI {
  private static normalizeRole(role: Role): Role {
    const rawPermissions = (role.permissions || []) as unknown[];
    return {
      ...role,
      permissions: rawPermissions
        .map(permission => {
          if (typeof permission === 'string') return permission;
          if (permission && typeof permission === 'object') {
            const item = permission as { code?: string; name?: string; id?: number };
            return item.code || item.name || String(item.id || '');
          }
          return '';
        })
        .filter(Boolean),
    };
  }

  // 获取角色列表
  static async getRoles(params?: GetRolesParams): Promise<RoleListResponse> {
    const normalizedParams = params
      ? {
          ...params,
          page_size: params.page_size || params.size,
          size: undefined,
        }
      : undefined;
    const response = await httpClient.get<RoleListResponse>('/api/v1/roles', normalizedParams);
    return {
      ...response,
      roles: (response.roles || []).map(role => this.normalizeRole(role)),
    };
  }

  // 获取单个角色
  static async getRole(id: number): Promise<Role> {
    const role = await httpClient.get<Role>(`/api/v1/roles/${id}`);
    return this.normalizeRole(role);
  }

  // 创建角色
  static async createRole(data: CreateRoleRequest): Promise<Role> {
    const role = await httpClient.post<Role>('/api/v1/roles', data);
    return this.normalizeRole(role);
  }

  // 更新角色
  static async updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
    const role = await httpClient.put<Role>(`/api/v1/roles/${id}`, data);
    return this.normalizeRole(role);
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

  // 获取权限目录，包含 ID、资源、动作等元数据，用于真实角色授权
  static async getPermissionCatalog(): Promise<PermissionCatalogItem[]> {
    const response = await httpClient.get<{
      permissions: string[];
      items?: PermissionCatalogItem[];
    }>('/api/v1/permissions');
    return (
      response.items ||
      response.permissions.map(code => {
        const [resource, action] = code.split(':');
        return {
          id: 0,
          code,
          name: code,
          resource: resource || '',
          action: action || '',
        };
      })
    );
  }

  // 初始化当前租户的默认权限字典
  static async initDefaultPermissions(): Promise<void> {
    await httpClient.post<void>('/api/v1/permissions/init');
  }

  // 分配权限给角色
  static async assignPermissions(roleId: number, permissionIds: number[]): Promise<void> {
    await httpClient.post(`/api/v1/roles/${roleId}/permissions`, { permission_ids: permissionIds });
  }
}
