/**
 * CMDB React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type {
  CIQuery,
  GraphQuery,
  ImpactAnalysisRequest,
} from '@/types/cmdb';

export const CMDB_KEYS = {
  all: ['cmdb'] as const,
  cis: () => [...CMDB_KEYS.all, 'cis'] as const,
  ciList: (query?: CIQuery) => [...CMDB_KEYS.cis(), 'list', query] as const,
  ciDetail: (id: string) => [...CMDB_KEYS.cis(), 'detail', id] as const,
  ciRelationships: (id: string) =>
    [...CMDB_KEYS.cis(), 'relationships', id] as const,
  ciChanges: (id: string) => [...CMDB_KEYS.cis(), 'changes', id] as const,
  ciHealth: (id: string) => [...CMDB_KEYS.cis(), 'health', id] as const,
  graph: (query: GraphQuery) => [...CMDB_KEYS.all, 'graph', query] as const,
  impactAnalysis: (ciId: string) =>
    [...CMDB_KEYS.all, 'impact-analysis', ciId] as const,
  ciTypes: () => [...CMDB_KEYS.all, 'ci-types'] as const,
  stats: () => [...CMDB_KEYS.all, 'stats'] as const,
  discovery: () => [...CMDB_KEYS.all, 'discovery'] as const,
};

// Query Hooks
export function useCIsQuery(query?: CIQuery) {
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

export function useRelationshipGraphQuery(
  query: GraphQuery,
  enabled = true
) {
  return useQuery({
    queryKey: CMDB_KEYS.graph(query),
    queryFn: () => CMDBApi.getRelationshipGraph(query),
    enabled: enabled && !!query.rootCI,
    staleTime: 60000,
  });
}

export function useImpactAnalysisQuery(
  request: ImpactAnalysisRequest,
  enabled = true
) {
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
    queryFn: () => CMDBApi.getCIChangeHistory(ciId, params),
    enabled: enabled && !!ciId,
    staleTime: 60000,
  });
}

export function useCIHealthQuery(ciId: string, enabled = true) {
  return useQuery({
    queryKey: CMDB_KEYS.ciHealth(ciId),
    queryFn: () => CMDBApi.getCIHealth(ciId),
    enabled: enabled && !!ciId,
    staleTime: 300000,
  });
}

export function useCMDBStatsQuery(params?: {
  startDate?: string;
  endDate?: string;
}) {
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
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof CMDBApi.updateCI>[1];
    }) => CMDBApi.updateCI(id, data),
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
    onSuccess: (result) => {
      message.success(`已批量创建 ${result.length} 个配置项`);
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useCreateRelationshipMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: CMDBApi.createRelationship,
    onSuccess: (result) => {
      message.success('关系已创建');
      queryClient.invalidateQueries({
        queryKey: CMDB_KEYS.ciRelationships(String((result as any).parent_id ?? (result as any).sourceCI)),
      });
      queryClient.invalidateQueries({
        queryKey: CMDB_KEYS.ciRelationships(String((result as any).child_id ?? (result as any).targetCI)),
      });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.stats() });
    },
  });
}

export function useDeleteRelationshipMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => CMDBApi.deleteRelationship(id),
    onSuccess: () => {
      message.success('关系已删除');
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.all });
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
  useRelationshipGraphQuery,
  useImpactAnalysisQuery,
  useCITypesQuery,
  useCIChangeHistoryQuery,
  useCIHealthQuery,
  useCMDBStatsQuery,
  useDiscoveryRulesQuery,
  useDiscoveryHistoryQuery,
  useCreateCIMutation,
  useUpdateCIMutation,
  useDeleteCIMutation,
  useBatchCreateCIsMutation,
  useCreateRelationshipMutation,
  useDeleteRelationshipMutation,
  useRunDiscoveryRuleMutation,
};

export default CMDBHooks;
