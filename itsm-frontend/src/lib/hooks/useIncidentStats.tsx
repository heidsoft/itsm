'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Siren, CircleDot, TriangleAlert, Clock3 } from 'lucide-react';

import { IncidentAPI } from '@/lib/api/incident-api';
import type { IncidentMetrics } from '@/lib/api/incident-api';
import type { PageStats } from '@/components/layout/BusinessPageTemplate';

export interface UseIncidentStatsResult {
  readonly stats: PageStats[];
  readonly loading: boolean;
  readonly metrics: IncidentMetrics | null;
  readonly refresh: () => Promise<void>;
}

/**
 * Loads incident metrics and maps them to the BusinessPageTemplate PageStats
 * array. Failure is non-fatal: the stats simply disappear from the toolbar
 * while the rest of the page keeps working.
 *
 * Metrics stay `null` on failure rather than falling back to an all-zero
 * object — showing "0 incidents" for a failed request would silently
 * misrepresent the system state.
 */
export function useIncidentStats(): UseIncidentStatsResult {
  const [metrics, setMetrics] = useState<IncidentMetrics | null>(null);
  const [loading, setLoading] = useState(false);

  // Guards against out-of-order responses: only the most recent request is
  // allowed to commit its result.
  const requestIdRef = useRef(0);

  const fetchStats = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const next = await IncidentAPI.getIncidentMetrics();
      if (requestIdRef.current !== requestId) return;
      setMetrics(next);
    } catch (error) {
      if (requestIdRef.current !== requestId) return;
      console.error('Failed to fetch incident stats:', error);
      setMetrics(null);
    } finally {
      if (requestIdRef.current === requestId) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void fetchStats();
  }, [fetchStats]);

  const stats = useMemo<PageStats[]>(
    () =>
      metrics
        ? [
            {
              label: '总事件数',
              value: metrics.totalIncidents || 0,
              color: '#3b82f6',
              icon: <Siren size={20} strokeWidth={1.8} />,
            },
            {
              label: '待处理',
              value: metrics.openIncidents || 0,
              color: '#faad14',
              icon: <CircleDot size={20} strokeWidth={1.8} />,
            },
            {
              label: '紧急事件',
              value: metrics.criticalIncidents || 0,
              color: '#ff4d4f',
              icon: <TriangleAlert size={20} strokeWidth={1.8} />,
            },
            {
              label: '平均解决时间',
              value: Math.round((metrics.avgResolutionTime || 0) / 60),
              suffix: '分钟',
              color: '#52c41a',
              icon: <Clock3 size={20} strokeWidth={1.8} />,
            },
          ]
        : [],
    [metrics]
  );

  return { stats, loading, metrics, refresh: fetchStats };
}
