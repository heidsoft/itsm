/**
 * useSLARefresh - SLA 数据自动刷新钩子测试
 */

import { renderHook, act } from '@testing-library/react';
import { useSLARefresh } from '../useSLARefresh';

// Mock 时间函数
const mockNow = new Date('2024-01-20T10:00:00Z').getTime();

describe('useSLARefresh', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(mockNow);
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('应该开始自动刷新', () => {
    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000, // 1 minute
        enabled: true,
      })
    );

    expect(result.current.isRefreshing).toBe(false);
    expect(result.current.lastRefresh).toBeNull();
  });

  it('应该按间隔刷新', async () => {
    const mockRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    // 等待第一次刷新
    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);
    expect(result.current.lastRefresh).toBeNear(new Date(mockNow + 60000));
  });

  it('应该禁用刷新', () => {
    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: false,
      })
    );

    // 即使计时器到达，也不应该刷新
    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(result.current.isRefreshing).toBe(false);
  });

  it('应该支持手动刷新', async () => {
    const mockRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    await act(async () => {
      await result.current.manualRefresh();
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);
  });

  it('应该显示刷新状态', () => {
    const mockRefresh = jest.fn(
      () => new Promise(resolve => setTimeout(resolve, 1000))
    );

    const { result, waitForNextUpdate } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    act(() => {
      result.current.manualRefresh();
    });

    expect(result.current.isRefreshing).toBe(true);

    // 等待 Promise 完成
    await waitForNextUpdate({ timeout: 2000 });

    expect(result.current.isRefreshing).toBe(false);
  });

  it('应该支持自定义间隔', () => {
    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 300000, // 5 minutes
        enabled: true,
      })
    );

    act(() => {
      jest.advanceTimersByTime(300000);
    });

    // 5 分钟后才刷新
    expect(result.current.lastRefresh).toBeNear(new Date(mockNow + 300000));
  });

  it('应该停止刷新', () => {
    const mockRefresh = jest.fn();

    const { result, unmount } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    // 先触发一次
    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);

    // 停止
    act(() => {
      result.current.stop();
    });

    // 再等待一段时间
    act(() => {
      jest.advanceTimersByTime(120000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1); // 不会再次调用
  });

  it('应该重置刷新计时器', async () => {
    const mockRefresh = jest.fn();

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    // 等待 30 秒
    act(() => {
      jest.advanceTimersByTime(30000);
    });

    // 手动刷新，重置计时器
    await act(async () => {
      await result.current.manualRefresh();
    });

    // 再过 30 秒不应该触发自动刷新（因为刚刷新过）
    act(() => {
      jest.advanceTimersByTime(30000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);

    // 再过 30 秒才应该触发
    act(() => {
      jest.advanceTimersByTime(30000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(2);
  });

  it('应该处理刷新错误', async () => {
    const mockRefresh = jest.fn().mockRejectedValue(new Error('Refresh failed'));

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
        onError: jest.fn(),
      })
    );

    await act(async () => {
      try {
        await result.current.manualRefresh();
      } catch (e) {
        // 错误应该被处理
      }
    });

    expect(result.current.error).toBe('Refresh failed');
    expect(result.current.isRefreshing).toBe(false);
  });

  it('应该在组件卸载时清理', () => {
    const mockRefresh = jest.fn();

    const { result, unmount } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    unmount();

    // 卸载后不应该有内存泄漏
    act(() => {
      jest.advanceTimersByTime(60000);
    });

    // 清理后不应该调用刷新
    expect(mockRefresh).toHaveBeenCalledTimes(0);
  });

  it('应该支持暂停和恢复', () => {
    const mockRefresh = jest.fn();

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    // 暂停
    act(() => {
      result.current.pause();
    });

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(0);

    // 恢复
    act(() => {
      result.current.resume();
    });

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);
  });

  it('应该记录刷新次数', async () => {
    const mockRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        onRefresh: mockRefresh,
      })
    );

    // 手动刷新多次
    await act(async () => {
      await result.current.manualRefresh();
    });

    await act(async () => {
      await result.current.manualRefresh();
    });

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(result.current.refreshCount).toBe(3); // 2 manual + 1 auto
  });

  it('应该检查是否需要刷新', () => {
    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
      })
    );

    const needsRefresh = result.current.needsRefresh();
    expect(typeof needsRefresh).toBe('boolean');
  });

  it('应该支持条件刷新', () => {
    const mockRefresh = jest.fn();
    let refreshCount = 0;

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        shouldRefresh: () => refreshCount < 2,
        onRefresh: () => {
          mockRefresh();
          refreshCount++;
        },
      })
    );

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(1);

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalledTimes(2);

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    // 条件返回 false，不再刷新
    expect(mockRefresh).toHaveBeenCalledTimes(2);
  });

  it('应该支持后台运行', () => {
    const mockRefresh = jest.fn();

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        runInBackground: true,
        onRefresh: mockRefresh,
      })
    );

    // 页面不可见时仍应刷新
    act(() => {
      result.current.setVisibility(false);
    });

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    expect(mockRefresh).toHaveBeenCalled();
  });

  it('应该不在后台运行时停止刷新', () => {
    const mockRefresh = jest.fn();

    const { result } = renderHook(() =>
      useSLARefresh({
        interval: 60000,
        enabled: true,
        runInBackground: false,
        onRefresh: mockRefresh,
      })
    );

    act(() => {
      result.current.setVisibility(false);
    });

    act(() => {
      jest.advanceTimersByTime(60000);
    });

    // 应该停止刷新
    expect(mockRefresh).not.toHaveBeenCalled();
  });
});
