/**
 * useSLAViolations Hook
 * 管理 SLA 违规数据的获取、过滤和选择
 */

import { useState, useCallback, useEffect } from 'react';
import { message } from 'antd';
import type { SLAViolation, SLAFilters, UseSLAViolationsReturn } from '../types';
import { fetchSLAViolations } from '../services/sla-monitor-service';
import { filterSLAViolations, resetFilters } from '../utils/sla-filter-utils';

export const useSLAViolations = (
  initialFilters?: Partial<SLAFilters>,
  autoLoad: boolean = true
): UseSLAViolationsReturn => {
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  const [filteredViolations, setFilteredViolations] = useState<SLAViolation[]>([]);
  const [filters, setFiltersState] = useState<SLAFilters>({
    status: '',
    severity: '',
    type: '',
    dateRange: null,
    search: '',
    ...initialFilters,
  });
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  /**
   * 加载违规数据
   */
  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchSLAViolations({
        status: filters.status || undefined,
        severity: filters.severity || undefined,
        sla_type: filters.type || undefined,
        start_date: filters.dateRange?.[0]?.format('YYYY-MM-DD'),
        end_date: filters.dateRange?.[1]?.format('YYYY-MM-DD'),
        search: filters.search || undefined,
      });
      setViolations(data);
    } catch (error) {
      message.error('加载 SLA 违规列表失败');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  /**
   * 应用过滤器
   */
  useEffect(() => {
    const filtered = filterSLAViolations(violations, filters);
    setFilteredViolations(filtered);
  }, [violations, filters]);

  /**
   * 初始加载
   */
  useEffect(() => {
    if (autoLoad) {
      refresh();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /**
   * 设置过滤器
   */
  const setFilters = useCallback((newFilters: SLAFilters) => {
    setFiltersState(newFilters);
  }, []);

  /**
   * 选择行
   */
  const selectRow = useCallback((keys: React.Key[]) => {
    setSelectedRowKeys(keys);
  }, []);

  /**
   * 清除选择
   */
  const clearSelection = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  return {
    violations,
    filteredViolations,
    filters,
    loading,
    selectedRowKeys,
    setFilters,
    refresh,
    selectRow,
    clearSelection,
  };
};
