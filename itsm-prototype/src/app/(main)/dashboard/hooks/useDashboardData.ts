'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { DashboardData } from '../types/dashboard.types';

export function useDashboardData() {
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30000); // 30秒
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  // 使用React Query获取Dashboard数据
  const {
    data,
    isLoading,
    error,
    refetch,
    isError,
  } = useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: async () => {
      try {
        return await DashboardAPI.getOverview();
      } catch (err) {
        console.warn('Dashboard API调用失败，将使用Mock数据:', err);
        // 返回 null，让组件使用 Mock 数据
        return null;
      }
    },
    refetchInterval: autoRefresh ? refreshInterval : false,
    staleTime: 10000, // 10秒内数据被认为是新鲜的
    retry: 1, // 减少重试次数，快速回退到 Mock 数据
    retryDelay: 1000,
    // 即使 API 失败也不抛出错误，而是返回 null
    throwOnError: false,
  });

  // 更新最后更新时间
  useEffect(() => {
    if (data) {
      setLastUpdated(new Date());
    }
  }, [data]);

  // 兼容旧的Mock数据结构 - 如果API还没实现，使用Mock数据
  useEffect(() => {
    if (isError && error) {
      console.warn('Dashboard API调用失败，考虑回退到Mock数据:', error);
    }
  }, [isError, error]);

  return {
    data: data || getMockData(), // 如果API失败，使用Mock数据
    loading: isLoading,
    error: error?.message ?? null,
    lastUpdated,
    autoRefresh,
    refreshInterval,
    refresh: refetch,
    setAutoRefresh,
    setRefreshInterval,
    isConnected: !isError,
    connectionStatus: isError ? '连接失败' : '已连接',
  };
}

// Mock数据作为回退方案（新增工单分析维度）
function getMockData(): DashboardData {
  return {
    // 扩展为8个KPI指标
    kpiMetrics: [
      // 第1行：工单数量指标
      {
        id: 'total-tickets',
        title: '总工单数',
        value: 1248,
        unit: '个',
        color: '#3b82f6',
        trend: 'up' as const,
        change: 12.5,
        changeType: 'increase' as const,
        description: '本月累计工单',
      },
      {
        id: 'pending-tickets',
        title: '待处理工单',
        value: 156,
        unit: '个',
        color: '#f59e0b',
        trend: 'down' as const,
        change: 8.3,
        changeType: 'decrease' as const,
        description: '需要立即处理',
        alert: 156 > 200 ? 'warning' : null,
      },
      {
        id: 'in-progress-tickets',
        title: '处理中工单',
        value: 89,
        unit: '个',
        color: '#06b6d4',
        trend: 'up' as const,
        change: 15.2,
        changeType: 'increase' as const,
        description: '正在处理中',
      },
      {
        id: 'completed-tickets',
        title: '已完成工单',
        value: 945,
        unit: '个',
        color: '#10b981',
        trend: 'up' as const,
        change: 18.5,
        changeType: 'increase' as const,
        description: '本月完成',
      },
      // 第2行：时间与质量指标
      {
        id: 'avg-first-response',
        title: '平均首次响应时间',
        value: 2.5,
        unit: '小时',
        color: '#8b5cf6',
        trend: 'down' as const,
        change: 0.8,
        changeType: 'decrease' as const,
        description: '响应速度提升',
        target: 4,
      },
      {
        id: 'avg-resolution',
        title: '平均解决时间',
        value: 4.8,
        unit: '小时',
        color: '#ec4899',
        trend: 'down' as const,
        change: 1.2,
        changeType: 'decrease' as const,
        description: '解决效率提升',
        target: 8,
      },
      {
        id: 'sla-compliance',
        title: 'SLA达成率',
        value: 92.5,
        unit: '%',
        color: '#10b981',
        trend: 'up' as const,
        change: 2.1,
        changeType: 'increase' as const,
        description: '服务水平提升',
        target: 95,
        alert: 92.5 >= 95 ? 'success' : 'warning',
      },
      {
        id: 'overdue-tickets',
        title: '超时工单',
        value: 18,
        unit: '个',
        color: '#ef4444',
        trend: 'up' as const,
        change: 3,
        changeType: 'increase' as const,
        description: 'SLA违规工单',
        alert: 'error',
      },
    ],
    // 工单趋势（扩展新建/完成/待处理维度）
    ticketTrend: [
      { date: '11-16', open: 120, inProgress: 80, resolved: 200, closed: 150, newTickets: 45, completedTickets: 38, pendingTickets: 120 },
      { date: '11-17', open: 135, inProgress: 75, resolved: 180, closed: 145, newTickets: 52, completedTickets: 41, pendingTickets: 131 },
      { date: '11-18', open: 150, inProgress: 90, resolved: 220, closed: 160, newTickets: 48, completedTickets: 45, pendingTickets: 134 },
      { date: '11-19', open: 140, inProgress: 85, resolved: 210, closed: 155, newTickets: 38, completedTickets: 47, pendingTickets: 125 },
      { date: '11-20', open: 130, inProgress: 70, resolved: 190, closed: 140, newTickets: 42, completedTickets: 52, pendingTickets: 115 },
      { date: '11-21', open: 145, inProgress: 95, resolved: 230, closed: 165, newTickets: 58, completedTickets: 49, pendingTickets: 124 },
      { date: '11-22', open: 156, inProgress: 89, resolved: 215, closed: 150, newTickets: 61, completedTickets: 55, pendingTickets: 130 },
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
    
    // 新增：工单类型分布
    typeDistribution: [
      { type: '事件', count: 562, percentage: 45, color: '#ef4444' },
      { type: '服务请求', count: 399, percentage: 32, color: '#3b82f6' },
      { type: '问题', count: 187, percentage: 15, color: '#f59e0b' },
      { type: '变更', count: 100, percentage: 8, color: '#10b981' },
    ],
    
    // 新增：响应时间分布
    responseTimeDistribution: [
      { timeRange: '0-1小时', count: 562, percentage: 45, avgTime: 0.5 },
      { timeRange: '1-4小时', count: 437, percentage: 35, avgTime: 2.3 },
      { timeRange: '4-8小时', count: 187, percentage: 15, avgTime: 5.8 },
      { timeRange: '>8小时', count: 62, percentage: 5, avgTime: 12.5 },
    ],
    
    // 新增：团队工作负载
    teamWorkload: [
      { assignee: '张三', ticketCount: 89, avgResponseTime: 2.1, completionRate: 94, activeTickets: 12 },
      { assignee: '李四', ticketCount: 76, avgResponseTime: 2.8, completionRate: 91, activeTickets: 9 },
      { assignee: '王五', ticketCount: 67, avgResponseTime: 3.2, completionRate: 88, activeTickets: 11 },
      { assignee: '赵六', ticketCount: 56, avgResponseTime: 2.5, completionRate: 92, activeTickets: 7 },
      { assignee: '孙七', ticketCount: 45, avgResponseTime: 3.5, completionRate: 85, activeTickets: 8 },
    ],
    
    // 新增：优先级分布
    priorityDistribution: [
      { priority: '紧急', count: 187, percentage: 15, color: '#ef4444' },
      { priority: '高', count: 312, percentage: 25, color: '#f59e0b' },
      { priority: '中', count: 562, percentage: 45, color: '#3b82f6' },
      { priority: '低', count: 187, percentage: 15, color: '#6b7280' },
    ],
    
    // 新增：高峰时段分析
    peakHours: [
      { hour: '00', count: 3, avgResponseTime: 8.5 },
      { hour: '01', count: 2, avgResponseTime: 9.2 },
      { hour: '02', count: 1, avgResponseTime: 10.0 },
      { hour: '03', count: 1, avgResponseTime: 11.5 },
      { hour: '04', count: 2, avgResponseTime: 10.8 },
      { hour: '05', count: 5, avgResponseTime: 8.2 },
      { hour: '06', count: 12, avgResponseTime: 5.5 },
      { hour: '07', count: 25, avgResponseTime: 3.8 },
      { hour: '08', count: 45, avgResponseTime: 2.5 },
      { hour: '09', count: 78, avgResponseTime: 1.8 },
      { hour: '10', count: 92, avgResponseTime: 1.5 },
      { hour: '11', count: 67, avgResponseTime: 2.2 },
      { hour: '12', count: 45, avgResponseTime: 3.1 },
      { hour: '13', count: 56, avgResponseTime: 2.8 },
      { hour: '14', count: 85, avgResponseTime: 1.9 },
      { hour: '15', count: 91, avgResponseTime: 1.6 },
      { hour: '16', count: 72, avgResponseTime: 2.4 },
      { hour: '17', count: 58, avgResponseTime: 2.9 },
      { hour: '18', count: 34, avgResponseTime: 4.2 },
      { hour: '19', count: 23, avgResponseTime: 5.5 },
      { hour: '20', count: 15, avgResponseTime: 6.8 },
      { hour: '21', count: 9, avgResponseTime: 7.5 },
      { hour: '22', count: 6, avgResponseTime: 8.2 },
      { hour: '23', count: 4, avgResponseTime: 8.9 },
    ],
    
    // 元数据
    metadata: {
      lastUpdated: new Date().toISOString(),
      dateRange: {
        start: '2024-11-16',
        end: '2024-11-22',
      },
      totalTickets: 1248,
    },
  };
}
