'use client';

/**
 * 侧栏菜单展开态（openKeys）持久化。
 *
 * 为什么需要独立 store 而不是组件内 state：
 * Sidebar 在「主菜单」与「管理功能」两处分别渲染 MenuItems，两处共享同一棵树
 * 的父级归属；状态放组件里会导致两个 Menu 各记一套、刷新全丢。放在 zustand
 * + persist 里，刷新/重进后仍保持用户上次展开的组。
 *
 * 只存「被展开的 submenu key」一个数组，key 由父项 path 推导（见
 * MenuItems.submenuKeyFor），与菜单语言/标题无关，改名不受影响。
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SidebarOpenKeysState {
  /** 已展开的子菜单 key 列表（受控 openKeys） */
  openKeys: string[];

  /** 用户手动展开/收起后整体回写（antd onOpenChange 给全量） */
  setOpenKeys: (keys: string[]) => void;

  /**
   * 把一组祖先 key 并入展开集合（路由变化时用）。
   *
   * 合并而不是覆盖：用户可能同时开着两个组（一边看工单分类、一边核对 SLA 模板），
   * 覆盖会让刚展开的组被收起。已经是展开态则不写 store，避免重复渲染。
   */
  revealGroupChain: (keys: string[]) => void;

  /** 退出登录/切换租户时清空，避免上一个人的菜单展开态串到下一个账号 */
  clear: () => void;
}

export const useSidebarOpenKeys = create<SidebarOpenKeysState>()(
  persist(
    (set, get) => ({
      openKeys: [],

      setOpenKeys: keys => {
        if (sameKeys(get().openKeys, keys)) return;
        set({ openKeys: keys });
      },

      revealGroupChain: keys => {
        if (keys.length === 0) return;
        const current = get().openKeys;
        const next = [...current];
        for (const key of keys) {
          if (!next.includes(key)) next.push(key);
        }
        if (sameKeys(current, next)) return;
        set({ openKeys: next });
      },

      clear: () => set({ openKeys: [] }),
    }),
    {
      name: 'itsm-sidebar-open-keys',
      // 只持久化 openKeys；方法不入 storage
      partialize: state => ({ openKeys: state.openKeys }),
    }
  )
);

function sameKeys(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((key, index) => key === b[index]);
}
