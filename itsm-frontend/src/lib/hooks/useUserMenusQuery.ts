'use client';

/**
 * 当前用户可见菜单的 React Query 缓存
 *
 * Sidebar、面包屑等位置都需要根据登录用户权限获取菜单。
 * 直接用 useState + useEffect 会在多组件挂载时重复请求，集中到 React Query 后：
 *   - 默认 staleTime 5 分钟，期间重复挂载使用缓存
 *   - 所有调用方共享一份 in-flight Promise，不会并发触发
 *   - 业务方调用 invalidateUserMenus 即可强制刷新（菜单管理端 CRUD 后）
 *   - 监听 menu-api.ts 的 MENUS_UPDATED_EVENT 自动 invalidate，避免散弹多处重写
 *
 * 真正的运行时入口是 MenuController.GetUserMenus → /api/v1/auth/menus，
 * 菜单数据源由后端 seedMenus 写入数据库；前端不再有硬编码 fallback。
 */

import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { getUserMenus, MENUS_UPDATED_EVENT, type MenuTreeResponse } from '@/lib/api/menu-api';

export const USER_MENUS_KEY = ['auth', 'menus'] as const;

export interface UseUserMenusQueryOptions {
  /** 是否启用缓存请求。默认 true */
  enabled?: boolean;
  /** 缓存过期时间（毫秒）。默认 5 分钟 */
  staleTime?: number;
}

/**
 * 获取当前登录用户可见的菜单树（主菜单 + 管理菜单），带缓存。
 */
export function useUserMenusQuery(options: UseUserMenusQueryOptions = {}) {
  const { enabled = true, staleTime = 5 * 60 * 1000 } = options;
  return useQuery<MenuTreeResponse>({
    queryKey: USER_MENUS_KEY,
    queryFn: () => getUserMenus(),
    enabled,
    staleTime,
  });
}

/**
 * 主动失效用户菜单缓存，并在组件挂载期间自动监听 MENUS_UPDATED_EVENT。
 * 业务侧在菜单管理端 CRUD 成功后调用一次即可，无需关心事件绑定。
 */
export function useInvalidateUserMenus() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: USER_MENUS_KEY });

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const handler = () => invalidate();
    window.addEventListener(MENUS_UPDATED_EVENT, handler);
    return () => {
      window.removeEventListener(MENUS_UPDATED_EVENT, handler);
    };
    // invalidate 引用稳定，刻意省略依赖避免反复注册监听器
  }, []);

  return invalidate;
}
