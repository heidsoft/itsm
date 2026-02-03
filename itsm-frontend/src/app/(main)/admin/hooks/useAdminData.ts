'use client';

import { useState, useEffect } from 'react';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import { WorkflowAPI } from '@/lib/api/workflow-api';

export interface AdminStats {
  activeUsers: string | number;
  runningWorkflows: string | number;
  serviceCatalogItems: string | number;
  systemAlerts: string | number;
}

export const useAdminData = () => {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<AdminStats>({
    activeUsers: 0,
    runningWorkflows: 0,
    serviceCatalogItems: 0,
    systemAlerts: 0,
  });

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        // 并行请求数据
        const [userStats, workflowStats] = await Promise.allSettled([
          DashboardAPI.getUserStats(),
          WorkflowAPI.getWorkflows({ page: 1, pageSize: 1 }) // 只需获取总数，或者如果有专门的统计API更好
        ]);

        const newStats: Partial<AdminStats> = {};

        // 处理用户统计
        if (userStats.status === 'fulfilled') {
          newStats.activeUsers = userStats.value.active;
        }

        // 处理工作流统计
        if (workflowStats.status === 'fulfilled') {
          newStats.runningWorkflows = workflowStats.value.total; // 这里暂时用总数，因为getWorkflows返回列表
          // 如果有专门的 running instances 统计会更准确
          // 注意：WorkflowAPI.getInstances() 可能更适合获取运行中的实例数
        }

        // 尝试获取运行中的工作流实例数
        try {
            const instances = await WorkflowAPI.getInstances({ status: 'running', pageSize: 1 });
            newStats.runningWorkflows = instances.total;
        } catch (e) {
            console.warn('Failed to fetch running workflow instances', e);
        }

        setStats(prev => ({
          ...prev,
          ...newStats,
          // 这里的其他数据暂时没有现成的API，保持默认值或模拟值
          // serviceCatalogItems: 89, 
          // systemAlerts: 2
        }));

      } catch (error) {
        console.error('Failed to fetch admin data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  return { loading, stats };
};
