/**
 * API 服务层模板 - 快速创建 CRUD API 服务
 */

import { httpClient } from '@/lib/api/http-client';

// ============ 模板宏定义 ============
// 复制以下代码到新文件，替换 MODULE_NAME 和实体名

/*
import { httpClient } from '@/lib/api/http-client';

interface EntityName {
  id: number;
  title: string;
  status: string;
  created_at: string;
}

export const EntityNameApi = {
  // 列表查询
  list: (params?: { page?: number; pageSize?: number; keyword?: string }) =>
    httpClient.get<{ list: EntityName[]; total: number }>('/entity-name', params),

  // 详情
  get: (id: number) =>
    httpClient.get<EntityName>(`/entity-name/${id}`),

  // 创建
  create: (data: Partial<EntityName>) =>
    httpClient.post<EntityName>('/entity-name', data),

  // 更新
  update: (id: number, data: Partial<EntityName>) =>
    httpClient.put<EntityName>(`/entity-name/${id}`, data),

  // 删除
  delete: (id: number) =>
    httpClient.delete(`/entity-name/${id}`),

  // 批量删除
  batchDelete: (ids: number[]) =>
    httpClient.post('/entity-name/batch/delete', { ids }),

  // 状态更新
  updateStatus: (id: number, status: string) =>
    httpClient.put<EntityName>(`/entity-name/${id}/status`, { status }),
};
*/

// ============ 通用 CRUD ============

export function createCrudApi<T extends { id: number }>(
  endpoint: string,
  options?: {
    batchEndpoint?: string;
    customEndpoints?: Record<string, (id?: number, data?: unknown) => Promise<unknown>>;
  }
) {
  return {
    list: (params?: object) =>
      httpClient.get<{ list: T[]; total: number }>(endpoint, params),

    get: (id: number) =>
      httpClient.get<T>(`${endpoint}/${id}`),

    create: (data: Partial<T>) =>
      httpClient.post<T>(endpoint, data),

    update: (id: number, data: Partial<T>) =>
      httpClient.put<T>(`${endpoint}/${id}`, data),

    delete: (id: number) =>
      httpClient.delete(`${endpoint}/${id}`),

    batchDelete: (ids: number[]) =>
      httpClient.post(`${options?.batchEndpoint || endpoint + '/batch'}/delete`, { ids }),

    ...options?.customEndpoints,
  };
}

// ============ 带租户的 API ============
// 注意：HttpClient 已内置租户支持，无需额外传递 headers

export function createTenantApi<T extends { id: number }>(endpoint: string) {
  return {
    list: (params?: object) =>
      httpClient.get<{ list: T[]; total: number }>(endpoint, params),

    get: (id: number) =>
      httpClient.get<T>(`${endpoint}/${id}`),

    create: (data: Partial<T>) =>
      httpClient.post<T>(endpoint, data),

    update: (id: number, data: Partial<T>) =>
      httpClient.put<T>(`${endpoint}/${id}`, data),

    delete: (id: number) =>
      httpClient.delete(`${endpoint}/${id}`),
  };
}

// ============ 示例用法 ============
/*
const UserApi = createCrudApi<User>('/users', {
  batchEndpoint: '/users/batch',
  customEndpoints: {
    resetPassword: (id: number) =>
      httpClient.post(`/users/${id}/reset-password`),
    changeRole: (id: number, role: string) =>
      httpClient.put(`/users/${id}/role`, { role }),
  },
});

// 使用
const { data } = await UserApi.list({ page: 1, pageSize: 10 });
await UserApi.create({ name: '新用户' });
await UserApi.update(id, { name: '更新名称' });
*/
