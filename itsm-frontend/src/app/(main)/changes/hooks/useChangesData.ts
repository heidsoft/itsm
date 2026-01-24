'use client';

import { useState, useEffect, useCallback } from 'react';
import { Change, ChangeStats, changeService } from '@/lib/services/change-service';

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
  const [error, setError] = useState<string | null>(null);

  const fetchChanges = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await changeService.getChanges({
        page: pagination.current,
        pageSize: pagination.pageSize,
        status: filter === '全部' ? undefined : filter,
        search: searchText || undefined,
      });
      setChanges(response.changes || []);
      setPagination(prev => ({ ...prev, total: response.total }));
    } catch (err) {
      console.error('获取变更列表失败:', err);
      setError(err instanceof Error ? err.message : '获取变更列表失败');
      // 使用空数组作为降级
      setChanges([]);
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filter, searchText]);

  const fetchStats = useCallback(async () => {
    try {
      const statsData = await changeService.getChangeStats();
      setStats(statsData);
    } catch (err) {
      console.error('获取统计数据失败:', err);
      // 使用默认统计值作为降级
      setStats({
        total: 0,
        draft: 0,
        pending: 0,
        approved: 0,
        implementing: 0,
        completed: 0,
        cancelled: 0,
      });
    }
  }, []);

  useEffect(() => {
    fetchChanges();
    fetchStats();
  }, [fetchChanges, fetchStats]);

  const handleTableChange = (paginationInfo: {
    current?: number;
    pageSize?: number;
    total?: number;
  }) => {
    setPagination(prev => ({ ...prev, ...paginationInfo }));
  };

  const refresh = useCallback(() => {
    fetchChanges();
    fetchStats();
  }, [fetchChanges, fetchStats]);

  return {
    changes,
    loading,
    stats,
    error,
    filter,
    setFilter,
    searchText,
    setSearchText,
    pagination,
    handleTableChange,
    fetchChanges,
    refresh,
  };
};
