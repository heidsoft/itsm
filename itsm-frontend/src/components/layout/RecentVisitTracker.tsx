'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';

import { useRecentVisitStore } from '@/lib/store/recent-visit-store';
import { useUserMenusQuery } from '@/lib/hooks/useUserMenusQuery';

import { collectPathLabels, resolveVisitTitle, shouldTrackVisit } from './recent-visit-title';

/**
 * 记录最近访问的页面，供 Ctrl+K 搜索弹窗的「最近访问」列表展示。
 *
 * 标题来源：后端菜单（`/api/v1/auth/menus`）——与侧栏、面包屑同一份数据，
 * 不再维护第二份硬编码 pathMeta。菜单未加载完成时用 URL 段兜底即可，
 * 不为了标题而阻塞记录（顺序无关，下次进入会更新）。
 */
export const RecentVisitTracker: React.FC = () => {
  const pathname = usePathname();
  const addVisit = useRecentVisitStore(state => state.addVisit);
  const { data: menus } = useUserMenusQuery();

  useEffect(() => {
    if (!shouldTrackVisit(pathname)) return;
    // 标题从菜单单一事实源取（进入页面时读一次即可，菜单后到时下次进入会刷新）。
    // 这里刻意不把菜单放进依赖：menus 是每次渲染的新引用，放进去会让每个渲染
    // 周期都写一条记录，把「最近访问」刷成一堆重复项。
    const map = new Map<string, string>();
    if (menus) {
      collectPathLabels(menus.main, map);
      collectPathLabels(menus.admin, map);
    }
    addVisit({ path: pathname, title: resolveVisitTitle(pathname, map) });
  }, [pathname, addVisit, menus]);

  return null;
};
