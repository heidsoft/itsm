/**
 * 布局状态管理 Store
 * 统一管理侧边栏折叠状态
 */

import { create } from 'zustand';

interface LayoutState {
  /** 侧边栏折叠状态 */
  collapsed: boolean;

  /** 切换折叠状态 */
  toggleCollapsed: () => void;

  /** 设置折叠状态 */
  setCollapsed: (collapsed: boolean) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  collapsed: false,

  toggleCollapsed: () =>
    set((state) => ({
      collapsed: !state.collapsed,
    })),

  setCollapsed: (collapsed) => set({ collapsed }),
}));
