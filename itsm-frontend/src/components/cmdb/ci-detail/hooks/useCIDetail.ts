/**
 * useCIDetail Hook (P1-2)
 * - 详情/类型/影响/历史均通过 React Query 驱动：自动竞态/卸载/缓存/重试
 * - loadXxx 改为 refetch 包装器，保留旧调用方契约（tab 切换时手动触发）
 */

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { App } from 'antd';

import type { UseCIDetailReturn } from '../types';
import type { CIType } from '@/types/biz/cmdb';
import {
  useCIQuery,
  useCITypesQuery,
  useImpactAnalysisQuery,
  useCIChangeHistoryQuery,
} from '@/lib/hooks/useCMDB';
import type { ImpactAnalysisRequest } from '@/types/cmdb';

const HISTORY_PAGE_SIZE = 20;

export const useCIDetail = (): UseCIDetailReturn => {
  const { id } = useParams() as { id: string };
  const { message } = App.useApp();

  // React Query：CI 详情（自动竞态/卸载/缓存/重试）
  const ciQuery = useCIQuery(id);
  const typesQuery = useCITypesQuery();

  // React Query：影响分析 & 变更历史（懒加载由 enabled 控制；
  // tab 切换可触发 refetch，等同旧 loadXxx 语义）
  const impactRequest: ImpactAnalysisRequest = {
    ciId: id,
    analysisType: 'both',
    maxDepth: 3,
  };
  const impactQuery = useImpactAnalysisQuery(impactRequest, !!id);
  const [historyPage, setHistoryPage] = useState(1);
  const historyQuery = useCIChangeHistoryQuery(
    id,
    { page: historyPage, pageSize: HISTORY_PAGE_SIZE },
    !!id
  );

  // 错误提示只在失败后通知一次，不在 render 路径里产生副作用
  const detailFailed = ciQuery.isError && !ciQuery.data;
  useEffect(() => {
    if (detailFailed) {
      message.error('加载资产详情失败');
    }
  }, [detailFailed, message]);

  const ci = ciQuery.data ?? null;
  const types: CIType[] = typesQuery.data ?? [];

  // loadXxx 包装为 refetch，保留旧调用方契约（onClick/tab 切换）
  const loadDetail = async () => {
    await ciQuery.refetch();
  };
  const loadImpactAnalysis = async () => {
    await impactQuery.refetch();
  };
  const loadChangeHistory = async () => {
    await historyQuery.refetch();
  };
  const loadHistoryPage = async (page: number) => {
    setHistoryPage(page);
  };

  const typeInfo = types.find(t => t.id === ci?.ciTypeId);

  return {
    ci,
    types,
    loading: ciQuery.isLoading || (!!id && typesQuery.isLoading),
    impactAnalysis: (impactQuery.data as unknown as UseCIDetailReturn['impactAnalysis']) ?? null,
    impactLoading: impactQuery.isFetching,
    changeHistory: historyQuery.data ?? null,
    historyLoading: historyQuery.isFetching,
    historyError: historyQuery.isError,
    loadDetail,
    loadImpactAnalysis,
    loadChangeHistory,
    loadHistoryPage,
    typeInfo,
  };
};
