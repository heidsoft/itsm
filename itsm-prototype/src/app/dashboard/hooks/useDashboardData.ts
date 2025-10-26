'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import {
  DashboardData,
  DashboardState,
  DashboardConfig,
  UseDashboardDataReturn,
  DashboardWebSocketMessage,
} from '../types/dashboard.types';

// 模拟API服务
const dashboardApi = {
  // 获取仪表盘数据
  getDashboardData: async (): Promise<DashboardData> => {
    // 模拟API延迟
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    // 模拟数据
    return {
      kpiMetrics: [
        {
          id: 'total-tickets',
          title: '工单总数',
          value: 1247,
          change: 12.5,
          changeType: 'increase',
          trend: 'up',
          color: '#1890ff',
          description: '工单总数量',
        },
        {
          id: 'open-tickets',
          title: '待处理工单',
          value: 89,
          change: -8.2,
          changeType: 'decrease',
          trend: 'down',
          color: '#fa8c16',
          description: '当前待处理工单',
        },
        {
          id: 'resolved-tickets',
          title: '今日已解决',
          value: 156,
          change: 23.1,
          changeType: 'increase',
          trend: 'up',
          color: '#52c41a',
          description: '今日已解决工单',
        },
        {
          id: 'sla-compliance',
          title: 'SLA达成率',
          value: 94.2,
          unit: '%',
          change: 2.1,
          changeType: 'increase',
          trend: 'up',
          color: '#722ed1',
          description: '服务级别协议达成率',
        },
        {
          id: 'avg-resolution',
          title: '平均解决时间',
          value: 4.2,
          unit: '小时',
          change: -0.8,
          changeType: 'decrease',
          trend: 'down',
          color: '#13c2c2',
          description: '平均工单解决时间',
        },
        {
          id: 'user-satisfaction',
          title: '用户满意度',
          value: 4.6,
          unit: '/5',
          change: 0.2,
          changeType: 'increase',
          trend: 'up',
          color: '#eb2f96',
          description: '平均用户满意度评分',
        },
      ],
      ticketTrend: [
        { date: '2024-01-01', open: 120, inProgress: 45, resolved: 89, closed: 78 },
        { date: '2024-01-02', open: 135, inProgress: 52, resolved: 95, closed: 82 },
        { date: '2024-01-03', open: 142, inProgress: 48, resolved: 102, closed: 91 },
        { date: '2024-01-04', open: 128, inProgress: 55, resolved: 88, closed: 85 },
        { date: '2024-01-05', open: 145, inProgress: 49, resolved: 96, closed: 88 },
        { date: '2024-01-06', open: 138, inProgress: 51, resolved: 92, closed: 90 },
        { date: '2024-01-07', open: 132, inProgress: 47, resolved: 89, closed: 87 },
      ],
      incidentDistribution: [
        { category: '硬件故障', count: 45, percentage: 35.2, color: '#ff4d4f' },
        { category: '软件问题', count: 38, percentage: 29.7, color: '#1890ff' },
        { category: '网络问题', count: 25, percentage: 19.5, color: '#52c41a' },
        { category: '安全问题', count: 12, percentage: 9.4, color: '#fa8c16' },
        { category: '其他', count: 8, percentage: 6.2, color: '#722ed1' },
      ],
      slaData: [
        { service: '事件响应', target: 95, actual: 96.2, status: 'met' },
        { service: '服务请求', target: 90, actual: 89.8, status: 'warning' },
        { service: '问题解决', target: 85, actual: 87.5, status: 'met' },
        { service: '变更实施', target: 80, actual: 78.9, status: 'breach' },
      ],
      satisfactionData: [
        { month: '1月', rating: 4.2, responses: 156 },
        { month: '2月', rating: 4.4, responses: 189 },
        { month: '3月', rating: 4.3, responses: 167 },
        { month: '4月', rating: 4.5, responses: 198 },
        { month: '5月', rating: 4.6, responses: 203 },
        { month: '6月', rating: 4.7, responses: 187 },
      ],
      recentActivities: [
        {
          id: '1',
          type: 'ticket',
          title: '工单-2024-001',
          description: '用户无法访问邮件系统',
          user: '张三',
          timestamp: '2024-01-07T10:30:00Z',
          status: '处理中',
          priority: 'high',
        },
        {
          id: '2',
          type: 'incident',
          title: '事件-2024-045',
          description: '生产环境服务器宕机',
          user: '李四',
          timestamp: '2024-01-07T09:15:00Z',
          status: '已解决',
          priority: 'urgent',
        },
        {
          id: '3',
          type: 'change',
          title: '变更-2024-012',
          description: '数据库迁移到新服务器',
          user: '王五',
          timestamp: '2024-01-07T08:45:00Z',
          status: '已批准',
          priority: 'medium',
        },
        {
          id: '4',
          type: 'problem',
          title: '问题-2024-008',
          description: '重复问题根本原因分析',
          user: '赵六',
          timestamp: '2024-01-07T07:20:00Z',
          status: '调查中',
          priority: 'high',
        },
      ],
      quickActions: [
        {
          id: 'create-ticket',
          title: '创建工单',
          description: '创建新的支持工单',
          icon: '🎫',
          color: '#1890ff',
          path: '/tickets/create',
          permission: 'ticket:create',
        },
        {
          id: 'create-incident',
          title: '报告事件',
          description: '报告严重事件',
          icon: '🚨',
          color: '#ff4d4f',
          path: '/incidents/new',
          permission: 'incident:create',
        },
        {
          id: 'create-change',
          title: '申请变更',
          description: '提交变更请求',
          icon: '🔄',
          color: '#52c41a',
          path: '/changes/new',
          permission: 'change:create',
        },
        {
          id: 'view-reports',
          title: '查看报告',
          description: '访问系统报告',
          icon: '📊',
          color: '#722ed1',
          path: '/reports',
          permission: 'report:view',
        },
      ],
    };
  },
};

// WebSocket连接管理
class DashboardWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 5000;
  private listeners: ((message: DashboardWebSocketMessage) => void)[] = [];

  connect() {
    try {
      // 模拟WebSocket连接
      this.ws = new WebSocket('ws://localhost:8080/dashboard');
      
      this.ws.onopen = () => {
        console.log('Dashboard WebSocket connected');
        this.reconnectAttempts = 0;
      };

      this.ws.onmessage = (event) => {
        try {
          const message: DashboardWebSocketMessage = JSON.parse(event.data);
          this.listeners.forEach(listener => listener(message));
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      this.ws.onclose = () => {
        console.log('Dashboard WebSocket disconnected');
        this.handleReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('Dashboard WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
      this.handleReconnect();
    }
  }

  private handleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      setTimeout(() => {
        console.log(`Attempting to reconnect WebSocket (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        this.connect();
      }, this.reconnectInterval);
    }
  }

  addListener(listener: (message: DashboardWebSocketMessage) => void) {
    this.listeners.push(listener);
  }

  removeListener(listener: (message: DashboardWebSocketMessage) => void) {
    this.listeners = this.listeners.filter(l => l !== listener);
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// 仪表盘数据Hook
export const useDashboardData = (config?: Partial<DashboardConfig>): UseDashboardDataReturn => {
  const queryClient = useQueryClient();
  const wsRef = useRef<DashboardWebSocket | null>(null);
  
  const [autoRefresh, setAutoRefresh] = useState(config?.autoRefresh ?? true);
  const [refreshInterval, setRefreshInterval] = useState(config?.refreshInterval ?? 30000);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');

  // React Query配置
  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['dashboard-data'],
    queryFn: dashboardApi.getDashboardData,
    staleTime: 5 * 60 * 1000, // 5分钟缓存
    gcTime: 10 * 60 * 1000, // 10分钟垃圾回收
    refetchInterval: autoRefresh ? refreshInterval : false,
    refetchOnWindowFocus: true,
    retry: 3,
  });

  // WebSocket连接管理
  useEffect(() => {
    if (autoRefresh) {
      wsRef.current = new DashboardWebSocket();
      
      const messageHandler = (message: DashboardWebSocketMessage) => {
        if (message.type === 'dashboard_update') {
          queryClient.setQueryData(['dashboard-data'], (oldData: DashboardData | undefined) => {
            if (!oldData) return oldData;
            return { ...oldData, ...message.data };
          });
        }
      };

      wsRef.current.addListener(messageHandler);
      wsRef.current.connect();

      const checkConnection = () => {
        if (wsRef.current?.isConnected()) {
          setIsConnected(true);
          setConnectionStatus('connected');
        } else {
          setIsConnected(false);
          setConnectionStatus('disconnected');
        }
      };

      const interval = setInterval(checkConnection, 1000);

      return () => {
        clearInterval(interval);
        wsRef.current?.removeListener(messageHandler);
        wsRef.current?.disconnect();
      };
    }
  }, [autoRefresh, queryClient]);

  // 手动刷新
  const refresh = useCallback(async () => {
    try {
      await refetch();
      message.success('Dashboard data refreshed');
    } catch (error) {
      message.error('Failed to refresh dashboard data');
    }
  }, [refetch]);

  // 设置自动刷新
  const handleSetAutoRefresh = useCallback((enabled: boolean) => {
    setAutoRefresh(enabled);
  }, []);

  // 设置刷新间隔
  const handleSetRefreshInterval = useCallback((interval: number) => {
    setRefreshInterval(interval);
  }, []);

  return {
    // 数据
    data: data || null,
    loading: isLoading,
    error: error?.message || null,
    lastUpdated: data ? new Date().toISOString() : null,
    
    // 状态
    autoRefresh,
    refreshInterval,
    
    // 操作
    refresh,
    setAutoRefresh: handleSetAutoRefresh,
    setRefreshInterval: handleSetRefreshInterval,
    
    // 实时更新
    isConnected,
    connectionStatus,
  };
};
