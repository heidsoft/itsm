"use client";

import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocalStorage, useSessionStorage } from './usePerformance';

// 类型定义
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

// 带重试的API调用
const fetchWithRetry = async (retryCount = 0): Promise<DashboardData> => {
  try {
    // 模拟网络延迟
    await delay(Math.random() * 500 + 500); // 500-1000ms随机延迟
    
    // 模拟偶尔的网络错误
    if (Math.random() < 0.1 && retryCount === 0) {
      throw new Error('网络连接超时');
    }
  
    return {
      systemAlerts: [
        {
          type: "warning",
          message: "系统负载较高，建议检查服务器状态",
          time: "2分钟前",
          severity: "medium",
        },
      ],
    recentTickets: [
      {
        id: "T-2024-001",
        title: "网络连接异常",
        priority: "high",
        status: "processing",
        assignee: "张三",
        time: "10分钟前",
        category: "网络",
        sla: "4小时",
      },
      {
        id: "T-2024-002",
        title: "软件安装失败",
        priority: "medium",
        status: "pending",
        assignee: "李四",
        time: "30分钟前",
        category: "软件",
        sla: "8小时",
      },
      {
        id: "T-2024-003",
        title: "数据库性能优化",
        priority: "low",
        status: "resolved",
        assignee: "王五",
        time: "2小时前",
        category: "数据库",
        sla: "24小时",
      },
    ],
    recentActivities: [
      {
        operator: "张三",
        action: "处理了工单",
        target: "T-2024-001",
        time: "10分钟前",
        avatar: "https://api.dicebear.com/7x/avataaars/svg?seed=张三",
      },
      {
        operator: "系统",
        action: "自动分配工单",
        target: "T-2024-002",
        time: "30分钟前",
        avatar: "https://api.dicebear.com/7x/avataaars/svg?seed=系统",
      },
      {
        operator: "李四",
        action: "更新了配置",
        target: "数据库配置",
        time: "1小时前",
        avatar: "https://api.dicebear.com/7x/avataaars/svg?seed=李四",
      },
    ],
      kpiData: {
        totalTickets: { value: 1247 + Math.floor(Math.random() * 100), change: 12, trend: "up" },
        pendingEvents: { value: 23 + Math.floor(Math.random() * 10), change: -5, trend: "down" },
        activeUsers: { value: 156 + Math.floor(Math.random() * 20), change: 8, trend: "up" },
        avgResponseTime: { value: 2.4 + Math.random() * 0.5, change: -15, trend: "down" },
        slaCompliance: { value: 98.5 + Math.random() * 1, change: 2, trend: "up" },
        customerSatisfaction: { value: 4.7 + Math.random() * 0.3, change: 0.3, trend: "up" },
      },
    };
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