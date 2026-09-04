/**
 * CMDB React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CIRelationshipAPI } from '@/lib/api/cmdb-relationship';
import type { GetCIListRequest } from '@/lib/api/cmdb-api';
import type { GraphQuery, ImpactAnalysisRequest } from '@/types/cmdb';

export const CMDB_KEYS = {
  all: ['cmdb'] as const,
  cis: () => [...CMDB_KEYS.all, 'cis'] as const,
  ciList: (query?: GetCIListRequest) => [...CMDB_KEYS.cis(), 'list', query] as const,
  ciDetail: (id: string) => [...CMDB_KEYS.cis(), 'detail', id] as const,
  ciRelationships: (id: string) => [...CMDB_KEYS.cis(), 'relationships', id] as const,
  ciChanges: (id: string) => [...CMDB_KEYS.cis(), 'changes', id] as const,
  graph: (query: GraphQuery) => [...CMDB_KEYS.all, 'graph', query] as const,
  impactAnalysis: (ciId: string) => [...CMDB_KEYS.all, 'impact-analysis', ciId] as const,
  ciTypes: () => [...CMDB_KEYS.all, 'ci-types'] as const,
  stats: () => [...CMDB_KEYS.all, 'stats'] as const,
  discovery: () => [...CMDB_KEYS.all, 'discovery'] as const,
};

// Query Hooks
export function useCIsQuery(query?: GetCIListRequest) {
  return useQuery({
    queryKey: CMDB_KEYS.ciList(query),
    queryFn: () => CMDBApi.getCIs(query),
    staleTime: 60000,
  });
}

export function useCIQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: CMDB_KEYS.ciDetail(id),
    queryFn: () => CMDBApi.getCI(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useCIRelationshipsQuery(
  ciId: string,
  params?: { direction?: 'incoming' | 'outgoing' | 'both'; types?: string[] },
  enabled = true
) {
  return useQuery({
    queryKey: [...CMDB_KEYS.ciRelationships(ciId), params],
    queryFn: () => CMDBApi.getCIRelationships(ciId, params),
    enabled: enabled && !!ciId,
    staleTime: 60000,
  });
}

export function useRelationshipGraphQuery(query: GraphQuery, enabled = true) {
  return useQuery({
    queryKey: CMDB_KEYS.graph(query),
    queryFn: () => CMDBApi.getCITopology(Number(query.rootCI), query.depth),
    enabled: enabled && !!query.rootCI,
    staleTime: 60000,
  });
}

export function useImpactAnalysisQuery(request: ImpactAnalysisRequest, enabled = true) {
  return useQuery({
    queryKey: [...CMDB_KEYS.impactAnalysis(request.ciId), request],
    queryFn: () => CMDBApi.analyzeImpact(request),
    enabled: enabled && !!request.ciId,
    staleTime: 300000,
  });
}

export function useCITypesQuery() {
  return useQuery({
    queryKey: CMDB_KEYS.ciTypes(),
    queryFn: () => CMDBApi.getCITypes(),
    staleTime: 600000,
  });
}

// useRelationshipTypesQuery P1-2: 关系词表（来自后端单一源 /cmdb/relationship-types）
// 返回结构含 name/description/direction/reverse/icon，frontend/lib/cmdb/relationship-vocabulary.ts
// 进一步加工为本地缓存（fail-soft），避免重复请求。
export function useRelationshipTypesQuery() {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'relationship-types'] as const,
    queryFn: () => CMDBApi.getRelationshipTypes(),
    staleTime: 600000,
  });
}

// useRelationshipTypesV2Query P1-2: 走 CIRelationshipAPI.getRelationshipTypes
// （该接口对返回结构做了归一化处理：直接返回数组；用于关系管理 UI）
export function useRelationshipTypesV2Query() {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'relationship-types-v2'] as const,
    queryFn: () => CIRelationshipAPI.getRelationshipTypes(),
    staleTime: 600000,
  });
}

// useOntologyQuery P1-2: 本体自描述端点（version/ciTypes/relationshipTypes/enums/aiTools）
// 后端 /api/v1/cmdb/ontology，单一接口取所有 schema 元数据，避免前端维护多份副本。
export function useOntologyQuery() {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'ontology'] as const,
    queryFn: () => CMDBApi.getOntology(),
    staleTime: 600000,
  });
}

export function useCIChangeHistoryQuery(
  ciId: string,
  params?: {
    startDate?: string;
    endDate?: string;
    page?: number;
    pageSize?: number;
  },
  enabled = true
) {
  return useQuery({
    queryKey: [...CMDB_KEYS.ciChanges(ciId), params],
    queryFn: () => CMDBApi.getCIChangeHistory(Number(ciId), params),
    enabled: enabled && !!ciId,
    staleTime: 60000,
  });
}

export function useCMDBStatsQuery(params?: { startDate?: string; endDate?: string }) {
  return useQuery({
    queryKey: [...CMDB_KEYS.stats(), params],
    queryFn: () => CMDBApi.getCMDBStats(params),
    staleTime: 300000,
  });
}

export function useDiscoveryRulesQuery() {
  return useQuery({
    queryKey: [...CMDB_KEYS.discovery(), 'rules'],
    queryFn: () => CMDBApi.getDiscoveryRules(),
    staleTime: 300000,
  });
}

export function useDiscoveryHistoryQuery(ruleId?: string) {
  return useQuery({
    queryKey: [...CMDB_KEYS.discovery(), 'history', ruleId],
    queryFn: () => CMDBApi.getDiscoveryHistory(ruleId),
    staleTime: 60000,
  });
}

// P1-2: 出/入向关系列表（CI 关系管理 UI 用）
export function useCIRelationshipsListQuery(
  ciId: number,
  options?: { includeOutgoing?: boolean; includeIncoming?: boolean; activeOnly?: boolean },
  enabled = true
) {
  return useQuery({
    queryKey: [...CMDB_KEYS.ciRelationships(String(ciId)), 'list', options],
    queryFn: () => CIRelationshipAPI.getCIRelationships(ciId, options),
    enabled: enabled && !!ciId,
    staleTime: 60000,
  });
}

// P1-2: 可关联的候选 CI（创建关系时搜索源/目标）
export function useAvailableCIsQuery(ciId: number, search?: string, enabled = true) {
  return useQuery({
    queryKey: [...CMDB_KEYS.cis(), 'available', String(ciId), search],
    queryFn: () => CIRelationshipAPI.getAvailableCIs(ciId, search),
    enabled: enabled && !!ciId,
    staleTime: 30000,
  });
}

// P1-2: 拓扑图（环检测用）
export function useTopologyGraphQuery(ciId: number, depth = 5, enabled = true) {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'topology', ciId, depth],
    queryFn: () => CIRelationshipAPI.getTopologyGraph(ciId, depth),
    enabled: enabled && !!ciId,
    staleTime: 60000,
  });
}

// P1-2: 云资源 / 云服务 / 云账号（CI 表单的级联选择数据源）
export function useCloudResourcesQuery(params?: Record<string, unknown>) {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'cloud-resources', params],
    queryFn: () => CMDBApi.getCloudResources(params),
    staleTime: 300000,
  });
}

export function useCloudServicesQuery(provider?: string) {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'cloud-services', provider],
    queryFn: () => CMDBApi.getCloudServices(provider),
    staleTime: 300000,
  });
}

export function useCloudAccountsQuery() {
  return useQuery({
    queryKey: [...CMDB_KEYS.all, 'cloud-accounts'] as const,
    queryFn: () => CMDBApi.getCloudAccounts(),
    staleTime: 300000,
  });
}

// Mutation Hooks
export function useCreateCIMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: CMDBApi.createCI,
    onSuccess: () => {
      message.success('配置项已创建');
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useUpdateCIMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof CMDBApi.updateCI>[1] }) =>
      CMDBApi.updateCI(id, data),
    onSuccess: (_, variables) => {
      message.success('配置项已更新');
      queryClient.invalidateQueries({
        queryKey: CMDB_KEYS.ciDetail(variables.id),
      });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
    },
  });
}

export function useDeleteCIMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => CMDBApi.deleteCI(id),
    onSuccess: () => {
      message.success('配置项已删除');
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useBatchCreateCIsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: CMDBApi.batchCreateCIs,
    onSuccess: result => {
      message.success(`已批量创建 ${result.length} 个配置项`);
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useCreateRelationshipMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof CIRelationshipAPI.createRelationship>[0]) =>
      CIRelationshipAPI.createRelationship(data),
    onSuccess: result => {
      message.success('关系已创建');
      queryClient.invalidateQueries({
        queryKey: CMDB_KEYS.ciRelationships(
          String((result as any).sourceCiId ?? (result as any).parentId)
        ),
      });
      queryClient.invalidateQueries({
        queryKey: CMDB_KEYS.ciRelationships(
          String((result as any).targetCiId ?? (result as any).childId)
        ),
      });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'topology'] });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useUpdateRelationshipMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof CIRelationshipAPI.updateRelationship>[1] }) =>
      CIRelationshipAPI.updateRelationship(id, data),
    onSuccess: () => {
      message.success('关系已更新');
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'relationships'] });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'topology'] });
    },
  });
}

export function useDeleteRelationshipMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number | string) =>
      typeof id === 'number'
        ? CIRelationshipAPI.deleteRelationship(id)
        : CIRelationshipAPI.deleteRelationship(Number(id)),
    onSuccess: () => {
      message.success('关系已删除');
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'relationships'] });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'topology'] });
    },
  });
}

export function useRunDiscoveryRuleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ruleId: string) => CMDBApi.runDiscoveryRule(ruleId),
    onSuccess: () => {
      message.success('发现规则已启动');
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.discovery() });
    },
  });
}

const CMDBHooks = {
  useCIsQuery,
  useCIQuery,
  useCIRelationshipsQuery,
  useCIRelationshipsListQuery,
  useAvailableCIsQuery,
  useTopologyGraphQuery,
  useRelationshipGraphQuery,
  useImpactAnalysisQuery,
  useCITypesQuery,
  useCIChangeHistoryQuery,
  useCMDBStatsQuery,
  useDiscoveryRulesQuery,
  useDiscoveryHistoryQuery,
  useCreateCIMutation,
  useUpdateCIMutation,
  useDeleteCIMutation,
  useBatchCreateCIsMutation,
  useCreateRelationshipMutation,
  useUpdateRelationshipMutation,
  useDeleteRelationshipMutation,
  useRunDiscoveryRuleMutation,
  useRelationshipTypesV2Query,
  useCloudResourcesQuery,
  useCloudServicesQuery,
  useCloudAccountsQuery,
};

export default CMDBHooks;
