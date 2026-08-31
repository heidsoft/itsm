/**
 * Sidebar 模块导出
 *
 * 菜单完全由后端 /api/v1/auth/menus 提供（数据源：itsm-backend/pkg/seeder seedMenus），
 * 前端不再提供 getMenuConfig 静态配置。如需获取菜单，请使用
 * `@/lib/hooks/useUserMenusQuery` 共享 React Query 缓存。
 */

export { Sidebar } from './Sidebar';
export { MenuItems } from './MenuItems';
export { iconStyle, iconMap, getIconByName } from './icons';
