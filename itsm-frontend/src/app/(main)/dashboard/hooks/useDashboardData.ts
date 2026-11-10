'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { DashboardData, KPIMetric } from '../types/dashboard.types';

export function useDashboardData() {
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30000); // 30秒
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  // 使用React Query获取Dashboard数据
  const { data, isLoading, error, refetch, isError } = useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: async () => {
      // 后端 /dashboard/overview 是工作台数字的唯一来源：工单类 KPI 已按登录者的
      // 行级数据范围统计，与 /tickets 列表、/tickets/stats 同一规则。
      // 此前这里再用 /tickets/stats 覆盖卡片数值，而环比 chip 仍来自租户口径的 overview，
      // 同一组卡片混着两种范围（2026-10-20 生产实测：坐席"待处理 12"下钻后找不到这 12 张单）。
      const overview = await DashboardAPI.getOverview();
      return { ...getInitialData(), ...overview };
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
    connectionStatus: isError ? 'Connection failed' : 'Connected',
  };
}

// 初始化空数据
function getInitialData(): DashboardData {
  return {
    // 扩展为8个KPI指标
    kpiMetrics: [
      // 第1行：工单数量指标
      placeholderMetric('total-tickets', 'Total Tickets', 'items', '#3b82f6', 'Loading...'),
      placeholderMetric('pending-tickets', 'Pending work order', 'items', '#f59e0b', 'Loading...', {
        alert: null,
      }),
      placeholderMetric('in-progress-tickets', 'Work order in progress', 'items', '#06b6d4', 'Loading...'),
      placeholderMetric('completed-tickets', 'Work order completed', 'items', '#10b981', 'Loading...'),
      // 第2行：时间与质量指标
      placeholderMetric('avg-first-response', 'Average first response time', 'hours', '#8b5cf6', 'No data', {
        target: 4,
      }),
      placeholderMetric('avg-resolution', 'Avg Resolution Time', 'hours', '#ec4899', 'No data', {
        target: 8,
      }),
      placeholderMetric('sla-compliance', 'SLA achievement rate', '%', '#10b981', 'No data', {
        target: 95,
        alert: 'warning',
      }),
      placeholderMetric('overdue-tickets', 'Overdue', 'items', '#ef4444', 'No data', {
        alert: 'error',
      }),
    ],
    // 工单趋势
    ticketTrend: [],
    incidentDistribution: [],
    // SLA 是聚合对象（后端 SLAComplianceData），不是数组；后端未返回前留空。
    slaData: null,
    satisfactionData: [],
    quickActions: [
      {
        id: 'create-ticket',
        title: 'Create a work order',
        description: 'Quickly create new IT tickets',
        path: '/tickets/create',
        color: '#3b82f6',
      },
      {
        id: 'create-incident',
        title: 'Report incident',
        description: 'Reporting IT incidents and incidents',
        path: '/incidents/create',
        color: '#ef4444',
      },
      {
        id: 'create-change',
        title: 'Commit changes',
        description: 'Submit IT change request',
        path: '/changes/new',
        color: '#10b981',
      },
      {
        id: 'view-reports',
        title: 'View report',
        description: 'View system reports and analysis',
        path: '/reports',
        color: '#8b5cf6',
      },
    ],
    recentActivities: [],

    typeDistribution: [],

    responseTimeDistribution: [],

    teamWorkload: [],

    priorityDistribution: [],

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

// placeholderMetric 给出请求返回前的占位卡片。趋势一律 stable、环比一律不可比：
// 数字与箭头方向只由后端 DTO 填充，占位阶段显示 ↑/↓ 等于凭空造出一个趋势。
function placeholderMetric(
  id: string,
  title: string,
  unit: string,
  color: string,
  description: string,
  extra?: Partial<KPIMetric>,
): KPIMetric {
  return {
    id,
    title,
    value: 0,
    unit,
    color,
    trend: 'stable',
    change: 0,
    changeType: 'stable',
    comparable: false,
    description,
    ...extra,
  };
}
