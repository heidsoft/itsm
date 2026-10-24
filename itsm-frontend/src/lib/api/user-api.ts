import { httpClient } from './http-client';

// 用户相关的接口定义

/**
 * 主角色词表（封闭）。单一源 = 后端 domain/role 包（domain/role/role.go），
 * 退役值不得再出现在此类型或任何 UI 词表中。
 */
export type UserRole =
  | 'super_admin'
  | 'admin'
  | 'manager'
  | 'it_admin'
  | 'security_admin'
  | 'audit_admin'
  | 'sysadmin'
  | 'agent'
  | 'technician'
  | 'end_user';

/**
 * 主角色下拉选项（中文标签）。词表与后端 domain/role 包一一对应；
 * 组管理页成员选择器的角色筛选也复用此词表。
 */
export const PRIMARY_ROLE_OPTIONS: { value: UserRole; label: string }[] = [
  { value: 'end_user', label: '最终用户' },
  { value: 'agent', label: '服务台坐席' },
  { value: 'technician', label: '技术员' },
  { value: 'manager', label: '部门经理' },
  { value: 'it_admin', label: 'IT管理员' },
  { value: 'security_admin', label: '安全管理员' },
  { value: 'audit_admin', label: '审计管理员' },
  { value: 'sysadmin', label: '系统运维' },
  { value: 'admin', label: '系统管理员' },
  { value: 'super_admin', label: '超级管理员' },
];

export const PRIMARY_ROLE_LABEL: Record<string, string> =
  Object.fromEntries(PRIMARY_ROLE_OPTIONS.map(o => [o.value, o.label]));

export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  active: boolean;
  tenantId: number;
  /** 主角色（users.role 枚举，词表单一源=后端 domain/role） */
  role: string;
  /** RBAC 多角色 ID（user_roles M2M 边） */
  roleIds?: number[];
  /** RBAC 多角色名称（仅展示用） */
  roleNames?: string[];
  createdAt: string;
  updatedAt: string;
}

/**
 * 角色标签：主角色高亮，其余 RBAC 角色去重后追加。
 * roleNames 来自 user_roles M2M，主角色同时存在于两边，不去重会渲染两枚相同标签。
 */
export function resolveRoleTags(
  user: Pick<User, 'role' | 'roleNames'>
): { label: string; primary: boolean }[] {
  const primary = user.role ? (PRIMARY_ROLE_LABEL[user.role] ?? user.role) : '';
  const tags = primary ? [{ label: primary, primary: true }] : [];
  for (const name of user.roleNames ?? []) {
    if (name !== primary && !tags.some(t => t.label === name)) {
      tags.push({ label: name, primary: false });
    }
  }
  return tags;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  password: string;
  tenantId: number;
  /** 主角色；不传时后端默认 end_user */
  role?: string;
  /** RBAC 多角色 ID（写入 user_roles 边） */
  roleIds?: number[];
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
  name?: string;
  department?: string;
  phone?: string;
  /** 主角色变更（后端会吊销存量 token） */
  role?: string;
  /** RBAC 多角色替换集：非 undefined 时整体替换；[] 表示清空 */
  roleIds?: number[];
}

/**
 * 本人资料（GET /api/v1/auth/me 与 PUT /api/v1/account/profile 的响应契约）。
 *
 * 与管理面 User 的差别是自助面看不到也不得提交 role/roleIds/active：
 * 后端 DTO 白名单会丢弃这些字段，因此这里同样不声明，避免 UI 误以为可改。
 */
export interface OwnProfile {
  id: number;
  username: string;
  email: string;
  name: string;
  role: string;
  mspRole?: string;
  department: string;
  departmentId: number;
  phone: string;
  active: boolean;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
  permissions?: string[];
}

/** 自助更新本人资料请求（字段白名单与后端 dto.UpdateOwnProfileRequest 一致） */
export interface UpdateOwnProfileRequest {
  name: string;
  email: string;
  phone?: string;
  department?: string;
}

export interface ListUsersParams {
  page?: number;
  pageSize?: number;
  tenantId?: number;
  status?: string;
  department?: string;
  search?: string;
}

export interface PagedUsersResponse {
  items: User[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface UserStatsResponse {
  total: number;
  active: number;
  inactive: number;
}

export interface BatchUpdateUsersRequest {
  userIds: number[];
  action: 'activate' | 'deactivate' | 'department';
  department?: string;
}

export interface SearchUsersParams {
  keyword: string;
  tenantId?: number;
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

  // 重置用户密码（管理员语义：不校验旧密码）
  // ⚠️ 后端当前**未注册** PUT /users/:id/reset-password（仅存在于预检映射与
  // swagger 注释中），调用必然 404。自助改密请用下方 changeOwnPassword。
  static async resetPassword(id: number, newPassword: string): Promise<void> {
    await httpClient.put(`${this.baseURL}/${id}/reset-password`, { newPassword: newPassword });
  }

  // 自助修改本人密码（校验旧密码）
  // 与上面的管理员重置是两个权限面：此处只能改自己，且必须提供当前密码。
  // 成功后服务端会吊销该用户全部存量 access token，需重新登录。
  static async changeOwnPassword(oldPassword: string, newPassword: string): Promise<void> {
    await httpClient.put('/api/v1/account/change-password', { oldPassword, newPassword });
  }

  // 读取本人资料（登录即可，无需 user:read）。
  // /auth/me 挂在 r 层而非 tenant 分组，端用户也可访问；管理面的 GET /users/:id
  // 需要 user:read，端用户用它读自己会 403。
  static async getOwnProfile(): Promise<OwnProfile> {
    return httpClient.get<OwnProfile>('/api/v1/auth/me');
  }

  // 自助更新本人资料（姓名/邮箱/电话/部门）。
  // 不能用 updateUser（PUT /users/:id）：那是 user:write 的管理面语义，
  // 端用户保存自己的资料必然 403，而放开该码等于交出整个用户目录的写权限。
  static async updateOwnProfile(payload: UpdateOwnProfileRequest): Promise<OwnProfile> {
    return httpClient.put<OwnProfile>('/api/v1/account/profile', payload);
  }

  // 获取用户统计
  static async getUserStats(tenantId?: number): Promise<UserStatsResponse> {
    const params = tenantId ? { tenantId: tenantId } : {};
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
        'Content-Type': 'multipart/form-data',
      },
    });
  }
}
