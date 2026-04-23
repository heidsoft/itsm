/**
 * SLA 实时更新 Hook
 * 提供基于优先级的智能轮询策略，支持可见性检测和动态刷新间隔
 */

import { useEffect, useRef, useCallback, useState } from 'react';

/**
 * 基于优先级的刷新间隔配置（毫秒）
 * 高优先级工单刷新更频繁
 */
const PRIORITY_REFRESH_INTERVALS: Record<string, number> = {
  urgent: 5000, // 5秒 - 紧急工单
  critical: 5000, // 5秒 - 关键工单
  high: 10000, // 10秒 - 高优先级
  medium: 30000, // 30秒 - 中优先级
  low: 60000, // 60秒 - 低优先级
  default: 30000, // 30秒 - 默认
};

/**
 * SLA 状态风险等级
 */
export type SLARiskLevel = 'critical' | 'warning' | 'normal';

interface SLARealTimeConfig {
  /** 最高优先级（用于确定刷新间隔） */
  highestPriority?: string;
  /** 是否有接近超时的工单 */
  hasAtRiskTickets?: boolean;
  /** 自定义刷新间隔（覆盖默认策略） */
  customInterval?: number;
  /** 是否启用（可用于暂停刷新） */
  enabled?: boolean;
  /** 页面隐藏时是否继续刷新 */
  refreshOnHidden?: boolean;
}

interface SLARealTimeReturn {
  /** 是否正在刷新 */
  isRefreshing: boolean;
  /** 上次刷新时间 */
  lastRefresh: Date | null;
  /** 当前刷新间隔 */
  currentInterval: number;
  /** 手动刷新 */
  refreshNow: () => Promise<void>;
  /** 暂停自动刷新 */
  pause: () => void;
  /** 恢复自动刷新 */
  resume: () => void;
  /** 是否已暂停 */
  isPaused: boolean;
  /** 页面是否可见 */
  isPageVisible: boolean;
}

/**
 * 计算最佳刷新间隔
 */
function calculateOptimalInterval(
  priority?: string,
  hasAtRisk?: boolean
): number {
  // 如果有即将超时的工单，使用更短的间隔
  if (hasAtRisk) {
    return 5000; // 5秒
  }

  // 根据优先级选择间隔
  const interval = PRIORITY_REFRESH_INTERVALS[priority || ''] || PRIORITY_REFRESH_INTERVALS.default;
  return interval;
}

/**
 * SLA 实时更新 Hook
 *
 * @example
 * ```tsx
 * const { refreshNow, isRefreshing, currentInterval } = useSLARealTime({
 *   highestPriority: 'urgent',
 *   hasAtRiskTickets: true,
 * }, async () => {
 *   const data = await fetchSLAData();
 *   setSlaData(data);
 * });
 * ```
 */
export function useSLARealTime(
  config: SLARealTimeConfig,
  onRefresh: () => Promise<void>
): SLARealTimeReturn {
  const {
    highestPriority,
    hasAtRiskTickets = false,
    customInterval,
    enabled = true,
    refreshOnHidden = false,
  } = config;

  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [isPaused, setIsPaused] = useState(false);
  const [isPageVisible, setIsPageVisible] = useState(true);

  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const onRefreshRef = useRef(onRefresh);

  // 保持回调引用最新
  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);

  // 计算当前刷新间隔
  const currentInterval = customInterval ?? calculateOptimalInterval(highestPriority, hasAtRiskTickets);

  // 执行刷新
  const doRefresh = useCallback(async () => {
    if (isPaused || !enabled) return;

    // 如果页面不可见且不要求后台刷新，跳过
    if (!isPageVisible && !refreshOnHidden) return;

    setIsRefreshing(true);
    try {
      await onRefreshRef.current();
      setLastRefresh(new Date());
    } catch (error) {
      console.error('SLA refresh failed:', error);
    } finally {
      setIsRefreshing(false);
    }
  }, [isPaused, enabled, isPageVisible, refreshOnHidden]);

  // 手动刷新
  const refreshNow = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await onRefreshRef.current();
      setLastRefresh(new Date());
    } catch (error) {
      console.error('Manual SLA refresh failed:', error);
      throw error;
    } finally {
      setIsRefreshing(false);
    }
  }, []);

  // 暂停
  const pause = useCallback(() => {
    setIsPaused(true);
  }, []);

  // 恢复
  const resume = useCallback(() => {
    setIsPaused(false);
  }, []);

  // 监听页面可见性变化
  useEffect(() => {
    const handleVisibilityChange = () => {
      const visible = document.visibilityState === 'visible';
      setIsPageVisible(visible);

      // 页面重新可见时立即刷新
      if (visible && enabled && !isPaused) {
        doRefresh();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [enabled, isPaused, doRefresh]);

  // 设置定时器
  useEffect(() => {
    if (!enabled || isPaused) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    // 立即执行一次
    doRefresh();

    // 设置定时器
    intervalRef.current = setInterval(doRefresh, currentInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [enabled, isPaused, currentInterval, doRefresh]);

  return {
    isRefreshing,
    lastRefresh,
    currentInterval,
    refreshNow,
    pause,
    resume,
    isPaused,
    isPageVisible,
  };
}

/**
 * 获取 SLA 剩余时间的格式化显示
 */
export function formatSLARemainingTime(seconds: number): {
  text: string;
  riskLevel: SLARiskLevel;
  isOverdue: boolean;
} {
  const isOverdue = seconds < 0;
  const absSeconds = Math.abs(seconds);

  let text: string;
  let riskLevel: SLARiskLevel;

  if (isOverdue) {
    // 已超时
    if (absSeconds < 3600) {
      text = `超时 ${Math.ceil(absSeconds / 60)} 分钟`;
    } else if (absSeconds < 86400) {
      text = `超时 ${Math.floor(absSeconds / 3600)} 小时 ${Math.ceil((absSeconds % 3600) / 60)} 分钟`;
    } else {
      text = `超时 ${Math.floor(absSeconds / 86400)} 天`;
    }
    riskLevel = 'critical';
  } else if (absSeconds < 1800) {
    // 30分钟内
    text = `${Math.ceil(absSeconds / 60)} 分钟`;
    riskLevel = 'critical';
  } else if (absSeconds < 3600) {
    // 1小时内
    text = `${Math.ceil(absSeconds / 60)} 分钟`;
    riskLevel = 'warning';
  } else if (absSeconds < 86400) {
    // 24小时内
    const hours = Math.floor(absSeconds / 3600);
    const minutes = Math.ceil((absSeconds % 3600) / 60);
    text = minutes > 0 ? `${hours} 小时 ${minutes} 分钟` : `${hours} 小时`;
    riskLevel = hours < 2 ? 'warning' : 'normal';
  } else {
    // 超过24小时
    text = `${Math.floor(absSeconds / 86400)} 天`;
    riskLevel = 'normal';
  }

  return { text, riskLevel, isOverdue };
}

/**
 * 获取 SLA 状态颜色
 */
export function getSLAStatusColor(riskLevel: SLARiskLevel): string {
  switch (riskLevel) {
    case 'critical':
      return '#cf1322'; // red
    case 'warning':
      return '#fa8c16'; // orange
    case 'normal':
      return '#52c41a'; // green
    default:
      return '#8c8c8c';
  }
}

export default useSLARealTime;
