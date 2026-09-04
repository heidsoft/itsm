/**
 * useCIDetail Hook (P1-2)
 * - 详情/类型/影响/历史均通过 React Query 驱动：自动竞态/卸载/缓存/重试
 * - loadXxx 改为 refetch 包装器，保留旧调用方契约（tab 切换时手动触发）
 */

import { useParams } from 'next/navigation';
import { message } from 'antd';

import type { UseCIDetailReturn } from '../types';
import type { CIType } from '@/types/biz/cmdb';
import {
  useCIQuery,
  useCITypesQuery,
  useImpactAnalysisQuery,
  useCIChangeHistoryQuery,
} from '@/lib/hooks/useCMDB';
import type { ImpactAnalysisRequest } from '@/types/cmdb';

export const useCIDetail = (): UseCIDetailReturn => {
  const { id } = useParams() as { id: string };

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
  const historyQuery = useCIChangeHistoryQuery(id, undefined, !!id);

  // 错误提示（保留旧 message.error 行为，仅在出错且已发生请求时弹一次）
  if (ciQuery.isError && !ciQuery.data) {
    message.error('加载资产详情失败');
  }

  const ci = ciQuery.data ?? null;
  const types: CIType[] = useMemoNormalizeTypes(typesQuery.data);

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

  const typeInfo = types.find(t => t.id === ci?.ciTypeId);

  return {
    ci,
    types,
    loading: ciQuery.isLoading || (!!id && typesQuery.isLoading),
    impactAnalysis: (impactQuery.data as unknown as UseCIDetailReturn['impactAnalysis']) ?? null,
    impactLoading: impactQuery.isFetching,
    changeHistory: (historyQuery.data as unknown as UseCIDetailReturn['changeHistory']) ?? null,
    historyLoading: historyQuery.isFetching,
    loadDetail,
    loadImpactAnalysis,
    loadChangeHistory,
    typeInfo,
  };
};

// useCITypesQuery 在 CIList 中已对返回值做兼容（data 直接是数组，或包在 {data,items}），
// 这里也按相同模式解析，避免单一 contract 假设。
function useMemoNormalizeTypes(raw: unknown): CIType[] {
  if (!raw) return [];
  const v = raw as { data?: CIType[]; items?: CIType[] } | CIType[];
  if (Array.isArray(v)) return v;
  if (Array.isArray(v.data)) return v.data;
  if (Array.isArray(v.items)) return v.items;
  return [];
}
