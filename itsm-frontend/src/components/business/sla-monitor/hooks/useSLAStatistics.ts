/**
 * useSLAStatistics Hook
 * 计算 SLA 违规统计数据
 */

import { useMemo, useCallback } from 'react';
import type { SLAStats, UseSLAStatisticsReturn, SLAViolation } from '../types';

export const useSLAStatistics = (): UseSLAStatisticsReturn => {
  /**
   * 计算统计数据
   */
  const calculateStats = useCallback((violations: SLAViolation[]): SLAStats => {
    const total = violations.length;
    const open = violations.filter(v => v.status === 'open').length;
    const resolved = violations.filter(v => v.status === 'resolved').length;
    const critical = violations.filter(v => v.severity === 'critical').length;

    return { total, open, resolved, critical };
  }, []);

  // 可以在这里添加派生统计数据
  const stats = useMemo<SLAStats>(() => {
    // 将在组件中传入 violations 并计算
    return { total: 0, open: 0, resolved: 0, critical: 0 };
  }, []);

  return {
    stats,
    calculateStats,
  };
};

/**
 * 辅助 Hook：根据传入的 violations 动态计算 stats
 */
export const useSLAStatisticsFrom = (
  violations: SLAViolation[]
): { stats: SLAStats } => {
  const { calculateStats } = useSLAStatistics();
  const stats = useMemo(() => calculateStats(violations), [violations, calculateStats]);

  return { stats };
};
