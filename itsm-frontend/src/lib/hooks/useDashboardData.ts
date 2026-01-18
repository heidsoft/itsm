"use client";

import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocalStorage, useSessionStorage } from './usePerformance';
import { DashboardAPI } from '@/lib/api/dashboard-api';

// 类型定义 - 与后端DashboardOverview对应
export interface SystemAlert {
  type: 'warning' | 'error' | 'info' | 'success';
  message: string;
  time: string;
  severity: 'low' | 'medium' | 'high';
}

export interface RecentTicket {
  id: string;
  title: string;
  priority: 'high' | 'medium' | 'low';
  status: 'processing' | 'pending' | 'resolved';
  assignee: string;
  time: string;
  category: string;
  sla: string;
}

export interface RecentActivity {
  operator: string;
  action: string;
  target: string;
  time: string;
  avatar: string;
}

export interface KPIData {
  totalTickets: { value: number; change: number; trend: 'up' | 'down' };
  pendingEvents: { value: number; change: number; trend: 'up' | 'down' };
  activeUsers: { value: number; change: number; trend: 'up' | 'down' };
  avgResponseTime: { value: number; change: number; trend: 'up' | 'down' };
  slaCompliance: { value: number; change: number; trend: 'up' | 'down' };
  customerSatisfaction: { value: number; change: number; trend: 'up' | 'down' };
}

export interface DashboardData {
  systemAlerts: SystemAlert[];
  recentTickets: RecentTicket[];
  recentActivities: RecentActivity[];
  kpiData: KPIData;
}

// 缓存配置
const CACHE_KEY = 'dashboard_data_cache';
const CACHE_DURATION = 5 * 60 * 1000; // 5分钟缓存
const MAX_RETRY_ATTEMPTS = 3;
const RETRY_DELAY = 1000; // 1秒重试延迟

// 延迟函数
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

// 将后端数据转换为前端格式
const convertBackendData = (backendData: {
  kpiMetrics?: Array<{ id: string; title: string; value: string | number; unit: string; color: string; trend: string; change: number; changeType: string; description: string }>;
  recentActivities?: Array<{ id: string; type: string; title: string; description: string; user: string; timestamp: string; priority?: string; status: string }>;
  quickActions?: Array<{ id: string; title: string; description: string; path: string; color: string; permission?: string }>;
}): DashboardData => {
  // 从KPI指标构建kpiData
  const kpiMetrics = backendData.kpiMetrics || [];
  const kpiData: KPIData = {
    totalTickets: { value: 0, change: 0, trend: 'up' },
    pendingEvents: { value: 0, change: 0, trend: 'up' },
    activeUsers: { value: 0, change: 0, trend: 'up' },
    avgResponseTime: { value: 0, change: 0, trend: 'up' },
    slaCompliance: { value: 0, change: 0, trend: 'up' },
    customerSatisfaction: { value: 0, change: 0, trend: 'up' },
  };

  // 映射KPI指标
  kpiMetrics.forEach(metric => {
    const trend = metric.trend as 'up' | 'down';
    const changeType = metric.changeType as 'increase' | 'decrease' | 'stable';
    const changeValue = changeType === 'increase' ? metric.change : changeType === 'decrease' ? -metric.change : 0;
    const numValue = typeof metric.value === 'string' ? parseFloat(metric.value) : metric.value;

    switch (metric.id) {
      case 'total_tickets':
        kpiData.totalTickets = { value: numValue, change: changeValue, trend };
        break;
      case 'pending_incidents':
        kpiData.pendingEvents = { value: numValue, change: changeValue, trend };
        break;
      case 'active_users':
        kpiData.activeUsers = { value: numValue, change: changeValue, trend };
        break;
      case 'avg_response_time':
        kpiData.avgResponseTime = { value: numValue, change: changeValue, trend };
        break;
      case 'sla_compliance':
        kpiData.slaCompliance = { value: numValue, change: changeValue, trend };
        break;
      case 'customer_satisfaction':
        kpiData.customerSatisfaction = { value: numValue, change: changeValue, trend };
        break;
    }
  });

  // 构建recentActivities
  const recentActivities: RecentActivity[] = (backendData.recentActivities || []).map(activity => ({
    operator: activity.user,
    action: activity.title,
    target: activity.description,
    time: activity.timestamp,
    avatar: `https://api.dicebear.com/7x/avataaars/svg?seed=${activity.user}`,
  }));

  // 系统告警（从quickActions推断或使用默认）
  const systemAlerts: SystemAlert[] = [];

  // 模拟recentTickets（实际应从单独的API获取）
  const recentTickets: RecentTicket[] = [];

  return {
    systemAlerts,
    recentTickets,
    recentActivities,
    kpiData,
  };
};

// 带重试的API调用
const fetchWithRetry = async (retryCount = 0): Promise<DashboardData> => {
  try {
    // 调用真实API获取仪表盘概览数据
    const backendData = await DashboardAPI.getOverview();
    return convertBackendData(backendData);
  } catch (error) {
    if (retryCount < MAX_RETRY_ATTEMPTS - 1) {
      console.warn(`API调用失败，正在重试... (${retryCount + 1}/${MAX_RETRY_ATTEMPTS})`);
      await delay(RETRY_DELAY * (retryCount + 1)); // 递增延迟
      return fetchWithRetry(retryCount + 1);
    }
    throw error;
  }
};

// 主要的数据获取函数
const fetchDashboardData = async (): Promise<DashboardData> => {
  return fetchWithRetry();
};

export const useDashboardData = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [retryCount, setRetryCount] = useState(0);

  // 使用本地存储缓存数据
  const [cachedData, setCachedData] = useLocalStorage<{
    data: DashboardData;
    timestamp: number;
  } | null>(CACHE_KEY, null);

  // 使用会话存储记录刷新状态
  const [refreshState, setRefreshState] = useSessionStorage('dashboard_refresh_state', {
    lastRefresh: 0,
    autoRefreshEnabled: true,
  });

  const abortControllerRef = useRef<AbortController | null>(null);
  const autoRefreshIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const isInitialLoadRef = useRef(true);

  const loadData = useCallback(async (forceRefresh = false) => {
    // 取消之前的请求
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    // 检查缓存（仅在非强制刷新时）
    if (!forceRefresh && cachedData && !isInitialLoadRef.current) {
      const now = Date.now();
      const cacheAge = now - cachedData.timestamp;

      if (cacheAge < CACHE_DURATION) {
        console.log('使用缓存数据，缓存年龄:', Math.round(cacheAge / 1000), '秒');
        setData(cachedData.data);
        setLastUpdated(new Date(cachedData.timestamp));
        setLoading(false);
        setError(null);
        return;
      }
    }

    // 标记非初始加载
    isInitialLoadRef.current = false;

    try {
      setLoading(true);
      setError(null);
      setRetryCount(0);

      abortControllerRef.current = new AbortController();

      const dashboardData = await fetchDashboardData();

      // 检查请求是否被取消
      if (abortControllerRef.current?.signal.aborted) {
        return;
      }

      const now = Date.now();
      setData(dashboardData);
      setLastUpdated(new Date(now));

      // 更新缓存
      setCachedData({
        data: dashboardData,
        timestamp: now,
      });

      // 更新刷新状态
      setRefreshState(prev => ({
        ...prev,
        lastRefresh: now,
      }));

    } catch (err) {
      if (abortControllerRef.current?.signal.aborted) {
        return;
      }

      const errorMessage = err instanceof Error ? err.message : '加载数据失败';
      setError(errorMessage);
      setRetryCount(prev => prev + 1);

      // 如果有缓存数据，在错误时仍然显示缓存数据
      if (cachedData && !data) {
        console.warn('API调用失败，使用缓存数据:', errorMessage);
        setData(cachedData.data);
        setLastUpdated(new Date(cachedData.timestamp));
      }
    } finally {
      setLoading(false);
      abortControllerRef.current = null;
    }
  }, [cachedData, setCachedData, setRefreshState, data]);

  const refreshData = useCallback(() => {
    loadData(true); // 强制刷新，跳过缓存
  }, [loadData]);

  const toggleAutoRefresh = useCallback(() => {
    setRefreshState(prev => ({
      ...prev,
      autoRefreshEnabled: !prev.autoRefreshEnabled,
    }));
  }, [setRefreshState]);

  const clearCache = useCallback(() => {
    setCachedData(null);
    loadData(true);
  }, [setCachedData, loadData]);

  // 初始化数据加载
  useEffect(() => {
    loadData();
  }, [loadData]);

  // 自动刷新逻辑
  useEffect(() => {
    if (!refreshState.autoRefreshEnabled) {
      if (autoRefreshIntervalRef.current) {
        clearInterval(autoRefreshIntervalRef.current);
        autoRefreshIntervalRef.current = null;
      }
      return;
    }

    // 设置自动刷新，每5分钟刷新一次
    autoRefreshIntervalRef.current = setInterval(() => {
      loadData();
    }, 5 * 60 * 1000);

    return () => {
      if (autoRefreshIntervalRef.current) {
        clearInterval(autoRefreshIntervalRef.current);
        autoRefreshIntervalRef.current = null;
      }
    };
  }, [refreshState.autoRefreshEnabled, loadData]);

  // 清理函数
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      if (autoRefreshIntervalRef.current) {
        clearInterval(autoRefreshIntervalRef.current);
      }
    };
  }, []);

  return {
    data,
    loading,
    error,
    lastUpdated,
    refreshData,
    retryCount,
    autoRefreshEnabled: refreshState.autoRefreshEnabled,
    toggleAutoRefresh,
    clearCache,
    // 计算缓存状态
    cacheStatus: cachedData ? {
      hasCache: true,
      cacheAge: Date.now() - cachedData.timestamp,
      isStale: Date.now() - cachedData.timestamp > CACHE_DURATION,
    } : {
      hasCache: false,
      cacheAge: 0,
      isStale: false,
    },
  };
};

export default useDashboardData;