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
    connectionStatus: isError ? '连接失败' : '已连接',
  };
}

// 初始化空数据
function getInitialData(): DashboardData {
  return {
    // 扩展为8个KPI指标
    kpiMetrics: [
      // 第1行：工单数量指标
      placeholderMetric('total-tickets', '总工单数', '个', '#3b82f6', '加载中...'),
      placeholderMetric('pending-tickets', '待处理工单', '个', '#f59e0b', '加载中...', {
        alert: null,
      }),
      placeholderMetric('in-progress-tickets', '处理中工单', '个', '#06b6d4', '加载中...'),
      placeholderMetric('completed-tickets', '已完成工单', '个', '#10b981', '加载中...'),
      // 第2行：时间与质量指标
      placeholderMetric('avg-first-response', '平均首次响应时间', '小时', '#8b5cf6', '暂无数据', {
        target: 4,
      }),
      placeholderMetric('avg-resolution', '平均解决时间', '小时', '#ec4899', '暂无数据', {
        target: 8,
      }),
      placeholderMetric('sla-compliance', 'SLA达成率', '%', '#10b981', '暂无数据', {
        target: 95,
        alert: 'warning',
      }),
      placeholderMetric('overdue-tickets', '超时工单', '个', '#ef4444', '暂无数据', {
        alert: 'error',
      }),
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
        path: '/incidents/create',
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
