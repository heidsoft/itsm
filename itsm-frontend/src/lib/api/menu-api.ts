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
 * HTTP 接口统一使用 camelCase
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
 * 菜单导出项——与后端 MenuExportItem / seeder menuSpec 字段对齐
 */
export interface MenuExportItem {
  name: string;
  path: string;
  icon?: string;
  parentPath?: string;
  permissionCode?: string;
  sortOrder: number;
  description?: string;
}

/**
 * 菜单初始化 diff 报告
 */
export interface MenuInitDiffResponse {
  added: MenuExportItem[];
  unchanged: MenuExportItem[];
  totalAdded: number;
}

/**
 * 菜单管理 API
 * 后端路由：/api/v1/menus（tenant_id 通过 JWT claims 注入）
 */
// 菜单数据变更事件：管理端 CRUD 成功后派发，Sidebar 监听后重新拉取 /api/v1/auth/menus
export const MENUS_UPDATED_EVENT = 'itsm:menus-updated';

export function notifyMenusUpdated(): void {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(MENUS_UPDATED_EVENT));
  }
}

export class MenuAdminAPI {
  private static readonly baseUrl = '/api/v1/menus';

  /** 列表（管理员视图，包含禁用/隐藏项）。超管可传 tenantId 查看其他租户菜单 */
  static async list(tenantId?: number): Promise<MenuListResponse> {
    return httpClient.get<MenuListResponse>(
      tenantId != null ? `${this.baseUrl}?tenantId=${tenantId}` : `${this.baseUrl}`,
    );
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

  /** 导出菜单为 seeder 兼容格式（parentPath 替代 parentId） */
  static async export(): Promise<MenuExportItem[]> {
    return httpClient.get<MenuExportItem[]>(`${this.baseUrl}/export`);
  }

  /** 从基线补齐缺失菜单，返回 added/unchanged diff 报告 */
  static async initDefaults(): Promise<MenuInitDiffResponse> {
    return httpClient.post<MenuInitDiffResponse>(
      `${this.baseUrl}/init`,
      {},
    );
  }
}

// 兼容历史调用：保留旧的函数导出
export const getUserMenus = async (): Promise<MenuTreeResponse> => {
  return httpClient.get<MenuTreeResponse>('/api/v1/auth/menus');
};
