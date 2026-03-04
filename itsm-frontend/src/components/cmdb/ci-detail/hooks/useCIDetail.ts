/**
 * useCIDetail Hook
 * 管理 CI 详情页面的所有数据和状态
 */

import { useState, useEffect, useCallback } from 'react';
import { message } from 'antd';
import { useParams } from 'next/navigation';

import type { UseCIDetailReturn } from '../types';
import type { ConfigurationItem } from '@/types/biz/cmdb';
import type { CIType } from '@/types/biz/cmdb'; // Keep as any-compatible
import { fetchCIDetail, fetchCIImpactAnalysis, fetchCIChangeHistory } from '../services/ci-detail-service';

export const useCIDetail = (): UseCIDetailReturn => {
  const { id } = useParams() as { id: string };

  const [ci, setCi] = useState<ConfigurationItem | null>(null);
  const [types, setTypes] = useState<CIType[]>([]);
  const [loading, setLoading] = useState(true);

  const [impactAnalysis, setImpactAnalysis] = useState<null>(null);
  const [impactLoading, setImpactLoading] = useState(false);

  const [changeHistory, setChangeHistory] = useState<null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);

  /**
   * 加载 CI 详情
   */
  const loadDetail = useCallback(async () => {
    if (!id) return;

    setLoading(true);
    try {
      const { ci: ciData, types: typeData } = await fetchCIDetail(id);
      setCi(ciData);
      setTypes(typeData);
    } catch (error) {
      message.error('加载资产详情失败');
    } finally {
      setLoading(false);
    }
  }, [id]);

  /**
   * 加载影响分析
   */
  const loadImpactAnalysis = useCallback(async () => {
    if (!id) return;

    const isMounted = true; // 简化处理，实际应使用 useEffect cleanup
    setImpactLoading(true);
    try {
      const data = await fetchCIImpactAnalysis(id);
      if (isMounted) {
        setImpactAnalysis(data);
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载影响分析失败，请稍后重试');
      }
    } finally {
      if (isMounted) {
        setImpactLoading(false);
      }
    }
  }, [id]);

  /**
   * 加载变更历史
   */
  const loadChangeHistory = useCallback(async () => {
    if (!id) return;

    const isMounted = true;
    setHistoryLoading(true);
    try {
      const data = await fetchCIChangeHistory(id);
      if (isMounted) {
        setChangeHistory(data);
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载变更历史失败，请稍后重试');
      }
    } finally {
      if (isMounted) {
        setHistoryLoading(false);
      }
    }
  }, [id]);

  /**
   * 初始加载
   */
  useEffect(() => {
    loadDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  /**
   * 计算类型信息
   */
  const typeInfo = types.find(t => t.id === ci?.ci_type_id);

  return {
    ci,
    types,
    loading,
    impactAnalysis,
    impactLoading,
    changeHistory,
    historyLoading,
    loadDetail,
    loadImpactAnalysis,
    loadChangeHistory,
    typeInfo,
  };
};
