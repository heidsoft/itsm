import { httpClient } from './http-client';
import type {
  Role,
  RoleListResponse,
  CreateRoleRequest,
  UpdateRoleRequest,
  GetRolesParams,
  PermissionCatalogItem,
  PermissionCatalogMeta,
  PermissionPolicyPayload,
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
    const { items, total, page, pageSize, totalPages } = await httpClient.get<RoleListResponse>(
      '/api/v1/roles',
      params,
    );
    return {
      items: items.map(role => this.normalizeRole(role)),
      total,
      page,
      pageSize,
      totalPages,
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

  // 获取所有权限码
  static async getPermissions(): Promise<string[]> {
    const response = await httpClient.get<{ items: PermissionCatalogItem[]; total: number }>('/api/v1/permissions');
    return response.items.map(item => item.code);
  }

  // 获取权限目录，包含 ID、资源、动作等元数据，用于真实角色授权
  static async getPermissionCatalog(): Promise<PermissionCatalogItem[]> {
    const response = await httpClient.get<{ items: PermissionCatalogItem[]; total: number }>('/api/v1/permissions');
    return response.items;
  }

  // 获取权限策略元数据（如 minApprovalsMax），对应 GET /api/v1/permissions/meta
  static async getPermissionCatalogMeta(): Promise<PermissionCatalogMeta> {
    return httpClient.get<PermissionCatalogMeta>('/api/v1/permissions/meta');
  }

  // 更新权限运行期策略（风险分级 / AI 主体授权 / 审批强度），对应 PUT /api/v1/permissions/:id。
  // 后端做闭集校验：允许 AI 执行中高风险必须开审批、高/危急必须双人。
  static async updatePermissionPolicy(
    id: number,
    data: { name?: string; description?: string; policy: PermissionPolicyPayload }
  ): Promise<PermissionCatalogItem> {
    return httpClient.put<PermissionCatalogItem>(`/api/v1/permissions/${id}`, data);
  }

  // 初始化当前租户的默认权限字典
  static async initDefaultPermissions(): Promise<void> {
    await httpClient.post<void>('/api/v1/permissions/init');
  }

  // 分配权限给角色
  static async assignPermissions(roleId: number, permissionIds: number[]): Promise<void> {
    await httpClient.post(`/api/v1/roles/${roleId}/permissions`, { permissionIds: permissionIds });
  }
}
