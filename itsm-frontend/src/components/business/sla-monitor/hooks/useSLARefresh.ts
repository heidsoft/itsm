/**
 * useSLARefresh Hook
 * 提供可配置的自动刷新功能
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import type { UseSLARefreshReturn } from '../types';

export const useSLARefresh = (): UseSLARefreshReturn => {
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  /**
   * 开始自动刷新
   */
  const startAutoRefresh = useCallback(
    (interval: number, callback: () => Promise<void>) => {
      stopAutoRefresh(); // 清除已有的

      const doRefresh = async () => {
        setIsRefreshing(true);
        try {
          await callback();
          setLastRefresh(new Date());
        } catch (error) {
          console.error('Auto refresh failed:', error);
        } finally {
          setIsRefreshing(false);
        }
      };

      // 立即执行一次
      doRefresh();

      // 设置定时器
      intervalRef.current = setInterval(doRefresh, interval);
    },
    []
  );

  /**
   * 停止自动刷新
   */
  const stopAutoRefresh = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  /**
   * 立即刷新一次
   */
  const refreshNow = useCallback(async (callback: () => Promise<void>) => {
    setIsRefreshing(true);
    try {
      await callback();
      setLastRefresh(new Date());
    } catch (error) {
      console.error('Manual refresh failed:', error);
      throw error;
    } finally {
      setIsRefreshing(false);
    }
  }, []);

  /**
   * 清理
   */
  useEffect(() => {
    return () => {
      stopAutoRefresh();
    };
  }, [stopAutoRefresh]);

  return {
    isRefreshing,
    lastRefresh,
    startAutoRefresh,
    stopAutoRefresh,
    refreshNow,
  };
};
