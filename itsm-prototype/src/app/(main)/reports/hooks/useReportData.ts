'use client';

import { useState, useEffect } from 'react';

interface ReportMetrics {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  slaCompliance: number;
  satisfactionScore: number;
  activeAgents: number;
  pendingTickets: number;
  urgentTickets: number;
}

export const useReportData = () => {
  const [metrics, setMetrics] = useState<ReportMetrics | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadReportMetrics();
  }, [timeRange]);

  const loadReportMetrics = async () => {
    try {
      setLoading(true);
      const mockMetrics: ReportMetrics = {
        totalTickets: 1247,
        resolvedTickets: 1189,
        avgResolutionTime: 8.7,
        slaCompliance: 87.3,
        satisfactionScore: 4.2,
        activeAgents: 12,
        pendingTickets: 58,
        urgentTickets: 23,
      };

      setMetrics(mockMetrics);
    } catch (error) {
      console.error('加载报表数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return { metrics, timeRange, setTimeRange, loading, loadReportMetrics };
};
