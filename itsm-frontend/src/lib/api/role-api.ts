import { httpClient } from './http-client';
import {
  Role,
  RoleListResponse,
  CreateRoleRequest,
  UpdateRoleRequest,
  GetRolesParams,
} from './api-config';

// Mock 数据用于后端未实现时回退
const MOCK_ROLES: Role[] = [
  {
    id: 1,
    name: '系统管理员',
    description: '拥有所有权限',
    permissions: ['*'],
    is_system: true,
    user_count: 2,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'IT支持',
    description: 'IT支持团队',
    permissions: ['tickets:*', 'incidents:*', 'knowledge:view'],
    is_system: false,
    user_count: 5,
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 3,
    name: '普通用户',
    description: '普通用户角色',
    permissions: ['tickets:create', 'tickets:view', 'knowledge:view'],
    is_system: false,
    user_count: 50,
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z',
  },
];

const MOCK_PERMISSIONS = [
  'dashboard:view',
  'tickets:view', 'tickets:create', 'tickets:update', 'tickets:delete',
  'incidents:view', 'incidents:create', 'incidents:update', 'incidents:delete',
  'problems:view', 'problems:create', 'problems:update', 'problems:delete',
  'changes:view', 'changes:create', 'changes:update', 'changes:delete',
  'service_catalog:view', 'service_catalog:create',
  'knowledge:view', 'knowledge:create', 'knowledge:update', 'knowledge:delete',
  'reports:view', 'reports:export',
  'cmdb:view', 'cmdb:create', 'cmdb:update', 'cmdb:delete',
  'admin:view', 'admin:users', 'admin:roles',
  'workflows:view', 'workflows:create', 'workflows:update',
];

export class RoleAPI {
  // 获取角色列表
  static async getRoles(params?: GetRolesParams): Promise<RoleListResponse> {
    try {
      return await httpClient.get<RoleListResponse>('/api/v1/roles', params);
    } catch {
      // 回退到 Mock 数据
      return {
        roles: MOCK_ROLES,
        total: MOCK_ROLES.length,
        page: 1,
        page_size: 10,
      };
    }
  }

  // 获取单个角色
  static async getRole(id: number): Promise<Role> {
    try {
      return await httpClient.get<Role>(`/api/v1/roles/${id}`);
    } catch {
      const role = MOCK_ROLES.find(r => r.id === id);
      if (!role) throw new Error('角色不存在');
      return role;
    }
  }

  // 创建角色
  static async createRole(data: CreateRoleRequest): Promise<Role> {
    try {
      return await httpClient.post<Role>('/api/v1/roles', data);
    } catch {
      // Mock 创建
      const newRole: Role = {
        id: Math.max(...MOCK_ROLES.map(r => r.id)) + 1,
        name: data.name,
        description: data.description || '',
        permissions: data.permissions || [],
        is_system: false,
        user_count: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      return newRole;
    }
  }

  // 更新角色
  static async updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
    try {
      return await httpClient.put<Role>(`/api/v1/roles/${id}`, data);
    } catch {
      const role = MOCK_ROLES.find(r => r.id === id);
      if (!role) throw new Error('角色不存在');
      return { ...role, ...data, updated_at: new Date().toISOString() };
    }
  }

  // 删除角色
  static async deleteRole(id: number): Promise<void> {
    try {
      await httpClient.delete<void>(`/api/v1/roles/${id}`);
    } catch {
      // Mock 删除静默成功
    }
  }

  // 获取所有权限列表
  static async getPermissions(): Promise<string[]> {
    try {
      const response = await httpClient.get<{ permissions: string[] }>('/api/v1/permissions');
      return response.permissions;
    } catch {
      return MOCK_PERMISSIONS;
    }
  }
}