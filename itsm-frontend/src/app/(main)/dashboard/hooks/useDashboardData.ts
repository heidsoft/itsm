'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import { ticketService } from '@/lib/services/ticket-service';
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
      // 并行请求 Dashboard 概览和真实的工单统计
      const [overviewResult, statsResult] = await Promise.allSettled([
        DashboardAPI.getOverview(),
        ticketService.getTicketStats()
      ]);

      let dashboardData = getInitialData(); // 默认使用空数据初始化

      // 处理 Dashboard Overview 数据
      if (overviewResult.status === 'fulfilled') {
        dashboardData = { ...dashboardData, ...overviewResult.value };
      } else {
        console.error('Dashboard Overview API调用失败:', overviewResult.reason);
        // 不再回退到 Mock 数据
      }

      // 处理真实工单统计数据（高优先级覆盖）
      if (statsResult.status === 'fulfilled') {
        const stats = statsResult.value;
        // 使用真实数据更新 KPI 指标
        dashboardData.kpiMetrics = dashboardData.kpiMetrics.map(metric => {
          switch (metric.id) {
            case 'total-tickets':
              return { ...metric, value: stats.total, description: '实时工单总数' };
            case 'pending-tickets':
              return { ...metric, value: stats.pending, description: '实时待处理' };
            case 'in-progress-tickets':
              return { ...metric, value: stats.in_progress, description: '实时处理中' };
            case 'completed-tickets':
              return { ...metric, value: stats.resolved + stats.closed, description: '实时已完成' };
            case 'overdue-tickets':
               return { ...metric, value: stats.overdue || 0, description: '实时超时工单' };
            default:
              return metric;
          }
        });
      } else {
        console.warn('Ticket Stats API调用失败:', statsResult.reason);
      }

      return dashboardData;
    },
    refetchInterval: autoRefresh ? refreshInterval : false,
    staleTime: 10000, // 10秒内数据被认为是新鲜的
    retry: 1,
    retryDelay: 1000,
  });

  // 更新最后更新时间
  useEffect(() => {
    if (data) {
      setLastUpdated(new Date());
    }
  }, [data]);

  return {
    data,
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

// 初始化空数据
function getInitialData(): DashboardData {
  return {
    // 扩展为8个KPI指标
    kpiMetrics: [
      // 第1行：工单数量指标
      {
        id: 'total-tickets',
        title: '总工单数',
        value: 0, 
        unit: '个',
        color: '#3b82f6',
        trend: 'up' as const,
        change: 0,
        changeType: 'increase' as const,
        description: '加载中...',
      },
      {
        id: 'pending-tickets',
        title: '待处理工单',
        value: 0,
        unit: '个',
        color: '#f59e0b',
        trend: 'down' as const,
        change: 0,
        changeType: 'decrease' as const,
        description: '加载中...',
        alert: null,
      },
      {
        id: 'in-progress-tickets',
        title: '处理中工单',
        value: 0,
        unit: '个',
        color: '#06b6d4',
        trend: 'up' as const,
        change: 0,
        changeType: 'increase' as const,
        description: '加载中...',
      },
      {
        id: 'completed-tickets',
        title: '已完成工单',
        value: 0,
        unit: '个',
        color: '#10b981',
        trend: 'up' as const,
        change: 0,
        changeType: 'increase' as const,
        description: '加载中...',
      },
      // 第2行：时间与质量指标
      {
        id: 'avg-first-response',
        title: '平均首次响应时间',
        value: 0,
        unit: '小时',
        color: '#8b5cf6',
        trend: 'down' as const,
        change: 0,
        changeType: 'decrease' as const,
        description: '暂无数据',
        target: 4,
      },
      {
        id: 'avg-resolution',
        title: '平均解决时间',
        value: 0,
        unit: '小时',
        color: '#ec4899',
        trend: 'down' as const,
        change: 0,
        changeType: 'decrease' as const,
        description: '暂无数据',
        target: 8,
      },
      {
        id: 'sla-compliance',
        title: 'SLA达成率',
        value: 0,
        unit: '%',
        color: '#10b981',
        trend: 'up' as const,
        change: 0,
        changeType: 'increase' as const,
        description: '暂无数据',
        target: 95,
        alert: 'warning',
      },
      {
        id: 'overdue-tickets',
        title: '超时工单',
        value: 0,
        unit: '个',
        color: '#ef4444',
        trend: 'up' as const,
        change: 0,
        changeType: 'increase' as const,
        description: '实时超时工单',
        alert: 'error',
      },
    ],
    // 工单趋势
    ticketTrend: [],
    incidentDistribution: [],
    slaData: [],
    satisfactionData: [],
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
    typeDistribution: [],
    
    // 新增：响应时间分布
    responseTimeDistribution: [],
    
    // 新增：团队工作负载
    teamWorkload: [],
    
    // 新增：优先级分布
    priorityDistribution: [],
    
    // 新增：高峰时段分析
    peakHours: [],
    
    // 元数据
    metadata: {
      lastUpdated: new Date().toISOString(),
      dateRange: {
        start: '',
        end: '',
      },
      totalTickets: 0,
    },
  };
}
