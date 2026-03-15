import { httpClient } from './http-client';

// 菜单项类型
export interface MenuItem {
  id: number;
  name: string;
  path: string;
  icon?: string;
  parent_id?: number;
  permission_code?: string;
  sort_order: number;
  tenant_id: number;
  is_visible: boolean;
  is_enabled: boolean;
  description?: string;
  children?: MenuItem[];
}

// 菜单树响应
export interface MenuTreeResponse {
  main: MenuItem[];
  admin: MenuItem[];
}

// 获取用户菜单 (后端路由: /api/v1/auth/menus)
export const getUserMenus = async (): Promise<MenuTreeResponse> => {
  return httpClient.get<MenuTreeResponse>('/api/v1/auth/menus');
};

// 获取菜单列表 (管理员用)
export const getMenus = async (tenantId: number): Promise<MenuItem[]> => {
  return httpClient.get<MenuItem[]>(`/api/v1/tenants/${tenantId}/menus`);
};

// 创建菜单
export const createMenu = async (tenantId: number, menu: Partial<MenuItem>): Promise<MenuItem> => {
  return httpClient.post<MenuItem>(`/api/v1/tenants/${tenantId}/menus`, menu);
};

// 更新菜单
export const updateMenu = async (tenantId: number, menuId: number, menu: Partial<MenuItem>): Promise<MenuItem> => {
  return httpClient.put<MenuItem>(`/api/v1/tenants/${tenantId}/menus/${menuId}`, menu);
};

// 删除菜单
export const deleteMenu = async (tenantId: number, menuId: number): Promise<void> => {
  return httpClient.delete(`/api/v1/tenants/${tenantId}/menus/${menuId}`);
};

// 初始化默认菜单
export const initDefaultMenus = async (tenantId: number): Promise<{ message: string; count: number }> => {
  return httpClient.post<{ message: string; count: number }>(`/api/v1/tenants/${tenantId}/menus/init`, {});
};
