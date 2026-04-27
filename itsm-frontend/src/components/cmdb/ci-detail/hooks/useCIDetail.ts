/**
 * useCIDetail Hook
 * 管理 CI 详情页面的所有数据和状态
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { message } from 'antd';
import { useParams } from 'next/navigation';

import type { UseCIDetailReturn, ImpactAnalysisData, ChangeHistoryData } from '../types';
import type { ConfigurationItem } from '@/types/biz/cmdb';
import type { CIType } from '@/types/biz/cmdb'; // Keep as any-compatible
import {
  fetchCIDetail,
  fetchCIImpactAnalysis,
  fetchCIChangeHistory,
} from '../services/ci-detail-service';

export const useCIDetail = (): UseCIDetailReturn => {
  const { id } = useParams() as { id: string };

  const [ci, setCi] = useState<any>(null);
  const [types, setTypes] = useState<CIType[]>([]);
  const [loading, setLoading] = useState(true);

  const [impactAnalysis, setImpactAnalysis] = useState<any>(null);
  const [impactLoading, setImpactLoading] = useState(false);

  const [changeHistory, setChangeHistory] = useState<any>(null);
  const [historyLoading, setHistoryLoading] = useState(false);

  // 用于取消异步操作的 ref
  const cancelledRef = useRef(false);

  // 组件卸载时设置取消标志
  useEffect(() => {
    return () => {
      cancelledRef.current = true;
    };
  }, []);

  /**
   * 加载 CI 详情
   */
  const loadDetail = useCallback(async () => {
    if (!id) return;

    setLoading(true);
    try {
      const { ci: ciData, types: typeData } = await fetchCIDetail(id);
      if (!cancelledRef.current) {
        setCi(ciData);
        setTypes(typeData);
      }
    } catch (error) {
      if (!cancelledRef.current) {
        message.error('加载资产详情失败');
      }
    } finally {
      if (!cancelledRef.current) {
        setLoading(false);
      }
    }
  }, [id]);

  /**
   * 加载影响分析
   */
  const loadImpactAnalysis = useCallback(async () => {
    if (!id) return;

    setImpactLoading(true);
    try {
      const data = await fetchCIImpactAnalysis(id);
      if (!cancelledRef.current) {
        setImpactAnalysis(data);
      }
    } catch (error) {
      if (!cancelledRef.current) {
        message.error('加载影响分析失败，请稍后重试');
      }
    } finally {
      if (!cancelledRef.current) {
        setImpactLoading(false);
      }
    }
  }, [id]);

  /**
   * 加载变更历史
   */
  const loadChangeHistory = useCallback(async () => {
    if (!id) return;

    setHistoryLoading(true);
    try {
      const data = await fetchCIChangeHistory(id);
      if (!cancelledRef.current) {
        setChangeHistory(data);
      }
    } catch (error) {
      if (!cancelledRef.current) {
        message.error('加载变更历史失败，请稍后重试');
      }
    } finally {
      if (!cancelledRef.current) {
        setHistoryLoading(false);
      }
    }
  }, [id]);

  /**
   * 初始加载
   */
  useEffect(() => {
    loadDetail();
  }, [loadDetail]);

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
