'use client';

import { useState, useEffect } from 'react';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';

export interface AdminStats {
  activeUsers: string | number | null;
  runningWorkflows: string | number | null;
  serviceCatalogItems: string | number | null;
  pendingTickets: string | number | null;
}

const EMPTY_STATS: AdminStats = {
  activeUsers: null,
  runningWorkflows: null,
  serviceCatalogItems: null,
  pendingTickets: null,
};

export const useAdminData = () => {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<AdminStats>(EMPTY_STATS);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      // 每张统计卡独立取数并降级：单个接口失败时该卡片显示 '—'，不影响其他卡片。
      const [userStats, workflowStats, instanceStats, catalogStats, ticketStats] =
        await Promise.allSettled([
          DashboardAPI.getUserStats(),
          WorkflowAPI.getWorkflows({ page: 1, pageSize: 1 }),
          WorkflowAPI.getInstances({ status: 'running', pageSize: 1 }),
          ServiceCatalogApi.getServices({ page: 1, pageSize: 1 }),
          DashboardAPI.getTicketStats(),
        ]);

      setStats({
        activeUsers: userStats.status === 'fulfilled' ? userStats.value.active : null,
        runningWorkflows:
          instanceStats.status === 'fulfilled'
            ? instanceStats.value.total
            : workflowStats.status === 'fulfilled'
              ? workflowStats.value.total
              : null,
        serviceCatalogItems:
          catalogStats.status === 'fulfilled' ? catalogStats.value.total : null,
        pendingTickets:
          ticketStats.status === 'fulfilled'
            ? ticketStats.value.open + ticketStats.value.inProgress
            : null,
      });
      setLoading(false);
    };

    fetchData();
  }, []);

  return { loading, stats };
};
