/**
 * Dashboard Data Hook
 * Provides unified data fetching for dashboard components
 */

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from './api-client';

interface DashboardStats {
  totalTickets: number;
  openTickets: number;
  resolvedToday: number;
  avgResolutionTime: number;
  slaCompliance: number;
  ticketsByPriority: Record<string, number>;
  ticketsByCategory: Record<string, number>;
  recentActivity: ActivityItem[];
}

interface ActivityItem {
  id: string;
  type: string;
  message: string;
  user: string;
  timestamp: string;
}

interface TicketTrend {
  date: string;
  created: number;
  resolved: number;
}

interface DashboardData {
  stats: DashboardStats;
  trends: TicketTrend[];
  topAssignees: { name: string; count: number }[];
}

export function useDashboardData() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch all dashboard data in parallel
      const [stats, trends, topAssignees] = await Promise.all([
        apiClient.get<DashboardStats>('/dashboard/stats'),
        apiClient.get<TicketTrend[]>('/dashboard/trends'),
        apiClient.get<{ name: string; count: number }[]>('/dashboard/top-assignees'),
      ]);

      setData({ stats, trends, topAssignees });
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch dashboard data'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  const refresh = useCallback(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  return {
    data,
    loading,
    error,
    refresh,
  };
}

// Hook for real-time dashboard updates
export function useRealtimeDashboard(intervalMs = 60000) {
  const { data, loading, error, refresh } = useDashboardData();

  useEffect(() => {
    const interval = setInterval(() => {
      refresh();
    }, intervalMs);

    return () => clearInterval(interval);
  }, [intervalMs, refresh]);

  return { data, loading, error, refresh };
}

export default useDashboardData;
