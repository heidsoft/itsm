/**
 * 布局状态 Store 测试
 * TDD: 先写测试，再实现
 */

import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { act } from '@testing-library/react';

// 测试将验证 Store 的行为
describe('useLayoutStore', () => {
  beforeEach(() => {
    // 重置模块以清除状态
    jest.resetModules();
  });

  describe('初始状态', () => {
    it('初始 collapsed 应为 false', async () => {
      // 动态导入以获取新鲜状态
      const { useLayoutStore } = await import('../layout-store');
      const state = useLayoutStore.getState();
      expect(state.collapsed).toBe(false);
    });
  });

  describe('toggleCollapsed', () => {
    it('应从 false 切换到 true', async () => {
      const { useLayoutStore } = await import('../layout-store');
      const { toggleCollapsed } = useLayoutStore.getState();

      act(() => {
        toggleCollapsed();
      });

      expect(useLayoutStore.getState().collapsed).toBe(true);
    });

    it('应从 true 切换到 false', async () => {
      const { useLayoutStore } = await import('../layout-store');

      // 先设置为 true
      act(() => {
        useLayoutStore.getState().setCollapsed(true);
      });

      // 再切换
      act(() => {
        useLayoutStore.getState().toggleCollapsed();
      });

      expect(useLayoutStore.getState().collapsed).toBe(false);
    });
  });

  describe('setCollapsed', () => {
    it('应设置 collapsed 为 true', async () => {
      const { useLayoutStore } = await import('../layout-store');
      const { setCollapsed } = useLayoutStore.getState();

      act(() => {
        setCollapsed(true);
      });

      expect(useLayoutStore.getState().collapsed).toBe(true);
    });

    it('应设置 collapsed 为 false', async () => {
      const { useLayoutStore } = await import('../layout-store');

      // 先设置为 true
      act(() => {
        useLayoutStore.getState().setCollapsed(true);
      });

      // 再设置为 false
      act(() => {
        useLayoutStore.getState().setCollapsed(false);
      });

      expect(useLayoutStore.getState().collapsed).toBe(false);
    });
  });

  describe('多消费者同步', () => {
    it('状态变化应同步到所有消费者', async () => {
      const { useLayoutStore } = await import('../layout-store');

      // 模拟多个消费者获取状态
      const consumer1 = useLayoutStore.getState();
      const consumer2 = useLayoutStore.getState();

      // 通过 consumer1 修改状态
      act(() => {
        consumer1.setCollapsed(true);
      });

      // consumer2 应看到更新
      expect(useLayoutStore.getState().collapsed).toBe(true);
    });
  });
});
