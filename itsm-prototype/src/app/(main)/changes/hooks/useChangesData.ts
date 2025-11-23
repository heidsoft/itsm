'use client';

import { useState, useEffect } from 'react';
import { Change, ChangeStats } from '@/lib/services/change-service';

export const useChangesData = () => {
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<ChangeStats>({
    total: 0,
    draft: 0,
    pending: 0,
    approved: 0,
    implementing: 0,
    completed: 0,
    cancelled: 0,
  });
  const [filter, setFilter] = useState('全部');
  const [searchText, setSearchText] = useState('');
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });

  const mockChanges: Change[] = [
    {
      id: 1,
      title: '系统升级变更',
      description: '升级核心系统版本',
      justification: '提升系统性能和安全性',
      status: 'pending',
      priority: 'high',
      type: 'standard',
      impactScope: 'high',
      riskLevel: 'medium',
      createdBy: 1,
      createdByName: '张三',
      assigneeName: '李四',
      tenantId: 1,
      plannedStartDate: '2024-01-15T09:00:00Z',
      plannedEndDate: '2024-01-15T18:00:00Z',
      implementationPlan: '系统升级实施计划',
      rollbackPlan: '系统回滚计划',
      createdAt: '2024-01-10T10:00:00Z',
      updatedAt: '2024-01-10T10:00:00Z',
    },
    {
      id: 2,
      title: '数据库配置变更',
      description: '调整数据库连接池配置',
      justification: '优化数据库性能',
      status: 'approved',
      priority: 'medium',
      type: 'normal',
      impactScope: 'medium',
      riskLevel: 'low',
      createdBy: 2,
      createdByName: '王五',
      assigneeName: '赵六',
      tenantId: 1,
      plannedStartDate: '2024-01-16T14:00:00Z',
      plannedEndDate: '2024-01-16T16:00:00Z',
      implementationPlan: '数据库配置变更计划',
      rollbackPlan: '数据库配置回滚计划',
      createdAt: '2024-01-12T14:30:00Z',
      updatedAt: '2024-01-12T14:30:00Z',
    },
  ];

  const mockStats: ChangeStats = {
    total: 156,
    draft: 12,
    pending: 23,
    approved: 45,
    implementing: 18,
    completed: 88,
    cancelled: 5,
  };

  const fetchChanges = async () => {
    try {
      setLoading(true);
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      setChanges(mockChanges);
      setPagination(prev => ({ ...prev, total: mockChanges.length }));
    } catch (error) {
      console.error('获取变更列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      setStats(mockStats);
    } catch (error) {
      console.error('获取统计数据失败:', error);
    }
  };

  useEffect(() => {
    fetchChanges();
    fetchStats();
  }, []);

  const handleTableChange = (paginationInfo: {
    current?: number;
    pageSize?: number;
    total?: number;
  }) => {
    setPagination(prev => ({ ...prev, ...paginationInfo }));
    fetchChanges();
  };

  return {
    changes,
    loading,
    stats,
    filter,
    setFilter,
    searchText,
    setSearchText,
    pagination,
    handleTableChange,
    fetchChanges,
  };
};
