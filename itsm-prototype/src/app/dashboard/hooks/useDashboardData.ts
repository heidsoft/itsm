'use client';

import { useState, useEffect, useCallback } from 'react';

export interface DashboardData {
  kpiMetrics: any[];
  ticketTrend: any[];
  incidentDistribution: any[];
  slaData: any[];
  satisfactionData: any[];
  quickActions: any[];
  recentActivities: any[];
}

export function useDashboardData() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30000); // 30秒
  const [isConnected, setIsConnected] = useState(true);
  const [connectionStatus, setConnectionStatus] = useState('已连接');

  // 模拟数据获取
  const fetchDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // 模拟API调用延迟
      await new Promise(resolve => setTimeout(resolve, 500));

      // 模拟数据
      const mockData: DashboardData = {
        kpiMetrics: [
          {
            id: 'total-tickets',
            title: '总工单数',
            value: 1248,
            unit: '',
            color: '#3b82f6',
            trend: 'up',
            change: 12.5,
            changeType: 'increase',
          },
          {
            id: 'open-tickets',
            title: '待处理工单',
            value: 156,
            unit: '',
            color: '#f59e0b',
            trend: 'down',
            change: 8.3,
            changeType: 'decrease',
          },
          {
            id: 'resolved-tickets',
            title: '已解决工单',
            value: 945,
            unit: '',
            color: '#10b981',
            trend: 'up',
            change: 15.2,
            changeType: 'increase',
          },
          {
            id: 'sla-compliance',
            title: 'SLA达成率',
            value: 92.5,
            unit: '%',
            color: '#8b5cf6',
            trend: 'up',
            change: 2.1,
            changeType: 'increase',
          },
          {
            id: 'avg-resolution',
            title: '平均解决时间',
            value: 4.5,
            unit: '小时',
            color: '#06b6d4',
            trend: 'down',
            change: 0.8,
            changeType: 'decrease',
          },
          {
            id: 'user-satisfaction',
            title: '用户满意度',
            value: 4.6,
            unit: '/5.0',
            color: '#ec4899',
            trend: 'up',
            change: 0.2,
            changeType: 'increase',
          },
        ],
        ticketTrend: [
          { date: '2024-01-01', open: 120, inProgress: 80, resolved: 200, closed: 150 },
          { date: '2024-01-02', open: 135, inProgress: 75, resolved: 180, closed: 145 },
          { date: '2024-01-03', open: 150, inProgress: 90, resolved: 220, closed: 160 },
          { date: '2024-01-04', open: 140, inProgress: 85, resolved: 210, closed: 155 },
          { date: '2024-01-05', open: 130, inProgress: 70, resolved: 190, closed: 140 },
          { date: '2024-01-06', open: 145, inProgress: 95, resolved: 230, closed: 165 },
          { date: '2024-01-07', open: 156, inProgress: 89, resolved: 215, closed: 150 },
        ],
        incidentDistribution: [
          { category: '网络故障', count: 45, color: '#ef4444' },
          { category: '系统故障', count: 32, color: '#f59e0b' },
          { category: '应用问题', count: 28, color: '#3b82f6' },
          { category: '硬件故障', count: 15, color: '#10b981' },
          { category: '其他', count: 10, color: '#6b7280' },
        ],
        slaData: [
          { service: '服务A', target: 95, actual: 96.5 },
          { service: '服务B', target: 98, actual: 97.2 },
          { service: '服务C', target: 90, actual: 92.1 },
        ],
        satisfactionData: [
          { month: '1月', rating: 4.2, responses: 120 },
          { month: '2月', rating: 4.4, responses: 135 },
          { month: '3月', rating: 4.5, responses: 150 },
          { month: '4月', rating: 4.6, responses: 145 },
        ],
        quickActions: [
          {
            id: 'create-ticket',
            title: '创建工单',
            description: '快速创建新的IT工单',
            path: '/tickets/create',
            color: '#3b82f6',
          },
          {
            id: 'create-incident',
            title: '报告事件',
            description: '报告IT事件和故障',
            path: '/incidents/new',
            color: '#ef4444',
          },
          {
            id: 'create-change',
            title: '提交变更',
            description: '提交IT变更请求',
            path: '/changes/new',
            color: '#10b981',
          },
          {
            id: 'view-reports',
            title: '查看报表',
            description: '查看系统报表和分析',
            path: '/reports',
            color: '#8b5cf6',
          },
        ],
        recentActivities: [],
      };

      setData(mockData);
      setLastUpdated(new Date());
      setIsConnected(true);
      setConnectionStatus('已连接');
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载数据失败');
      setIsConnected(false);
      setConnectionStatus('连接失败');
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  // 自动刷新
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      fetchDashboardData();
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, fetchDashboardData]);

  return {
    data,
    loading,
    error,
    lastUpdated,
    autoRefresh,
    refreshInterval,
    refresh: fetchDashboardData,
    setAutoRefresh,
    setRefreshInterval,
    isConnected,
    connectionStatus,
  };
}

