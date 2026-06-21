import { httpClient } from './http-client';

/**
 * 菜单数据传输对象
 * 与后端 dto.MenuDTO 字段保持一致（camelCase，由 http-client 自动转换）
 */
export interface MenuItem {
  id: number;
  name: string;
  path: string;
  icon?: string;
  parentId?: number | null;
  permissionCode?: string | null;
  sortOrder: number;
  tenantId: number;
  isVisible: boolean;
  isEnabled: boolean;
  description?: string;
  children?: MenuItem[];
}

/**
 * 菜单列表响应
 */
export interface MenuListResponse {
  menus: MenuItem[];
  total: number;
}

/**
 * 菜单树响应（当前登录用户可见的菜单）
 */
export interface MenuTreeResponse {
  main: MenuItem[];
  admin: MenuItem[];
}

/**
 * 创建/更新请求载荷
 * http-client 会自动将 camelCase 转为 snake_case，因此 parentId → parent_id
 */
export interface MenuRequest {
  name: string;
  path: string;
  icon?: string;
  parentId?: number | null;
  permissionCode?: string | null;
  sortOrder?: number;
  isVisible?: boolean;
  isEnabled?: boolean;
  description?: string;
}

/**
 * 菜单管理 API
 * 后端路由：/api/v1/menus（tenant_id 通过 JWT claims 注入）
 */
export class MenuAdminAPI {
  private static readonly baseUrl = '/api/v1/menus';

  /** 列表（管理员视图，包含禁用/隐藏项） */
  static async list(): Promise<MenuListResponse> {
    return httpClient.get<MenuListResponse>(this.baseUrl);
  }

  /** 详情 */
  static async get(id: number): Promise<MenuItem> {
    return httpClient.get<MenuItem>(`${this.baseUrl}/${id}`);
  }

  /** 创建 */
  static async create(payload: MenuRequest): Promise<MenuItem> {
    return httpClient.post<MenuItem>(this.baseUrl, payload);
  }

  /** 更新（局部字段） */
  static async update(id: number, payload: Partial<MenuRequest>): Promise<MenuItem> {
    return httpClient.put<MenuItem>(`${this.baseUrl}/${id}`, payload);
  }

  /** 删除 */
  static async remove(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }

  /** 重新初始化默认菜单（仅插入缺失项，已存在的不动） */
  static async initDefaults(): Promise<{ message: string; count: number }> {
    return httpClient.post<{ message: string; count: number }>(
      `${this.baseUrl}/init`,
      {},
    );
  }
}

// 兼容历史调用：保留旧的函数导出
export const getUserMenus = async (): Promise<MenuTreeResponse> => {
  return httpClient.get<MenuTreeResponse>('/api/v1/auth/menus');
};
