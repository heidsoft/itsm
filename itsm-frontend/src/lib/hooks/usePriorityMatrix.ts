/**
 * 优先级矩阵 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { PriorityMatrixApi } from '@/lib/api/priority-matrix-api';
import type {
  PriorityCalculationRequest,
  PriorityRuleQuery,
  PriorityAnalysisQuery,
} from '@/types/priority-matrix';

export const PRIORITY_MATRIX_KEYS = {
  all: ['priority-matrix'] as const,
  configs: () => [...PRIORITY_MATRIX_KEYS.all, 'configs'] as const,
  activeConfig: () => [...PRIORITY_MATRIX_KEYS.all, 'active-config'] as const,
  matrixData: (configId?: string) =>
    [...PRIORITY_MATRIX_KEYS.all, 'matrix-data', configId] as const,
  rules: (query?: PriorityRuleQuery) =>
    [...PRIORITY_MATRIX_KEYS.all, 'rules', query] as const,
  suggestion: (ticketId: number) =>
    [...PRIORITY_MATRIX_KEYS.all, 'suggestion', ticketId] as const,
  distribution: (query: PriorityAnalysisQuery) =>
    [...PRIORITY_MATRIX_KEYS.all, 'distribution', query] as const,
};

// Query Hooks
export function useMatrixConfigsQuery() {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.configs(),
    queryFn: () => PriorityMatrixApi.getMatrixConfigs(),
    staleTime: 300000,
  });
}

export function useActiveMatrixConfigQuery() {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.activeConfig(),
    queryFn: () => PriorityMatrixApi.getActiveMatrixConfig(),
    staleTime: 300000,
  });
}

export function useMatrixDataQuery(configId?: string) {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.matrixData(configId),
    queryFn: () => PriorityMatrixApi.getMatrixData(configId),
    staleTime: 60000,
  });
}

export function usePriorityRulesQuery(query?: PriorityRuleQuery) {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.rules(query),
    queryFn: () => PriorityMatrixApi.getPriorityRules(query),
    staleTime: 60000,
  });
}

export function usePrioritySuggestionQuery(ticketId: number, enabled = true) {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.suggestion(ticketId),
    queryFn: () => PriorityMatrixApi.getPrioritySuggestion(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 300000,
  });
}

export function usePriorityDistributionQuery(query: PriorityAnalysisQuery) {
  return useQuery({
    queryKey: PRIORITY_MATRIX_KEYS.distribution(query),
    queryFn: () => PriorityMatrixApi.getPriorityDistribution(query),
    staleTime: 300000,
  });
}

// Mutation Hooks
export function useCalculatePriorityMutation() {
  return useMutation({
    mutationFn: (request: PriorityCalculationRequest) =>
      PriorityMatrixApi.calculatePriority(request),
    onSuccess: () => {
      message.success('优先级计算完成');
    },
  });
}

export function useCreateMatrixConfigMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: PriorityMatrixApi.createMatrixConfig,
    onSuccess: () => {
      message.success('矩阵配置已创建');
      queryClient.invalidateQueries({ queryKey: PRIORITY_MATRIX_KEYS.configs() });
    },
  });
}

export function useActivateMatrixConfigMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (configId: string) => PriorityMatrixApi.activateMatrixConfig(configId),
    onSuccess: () => {
      message.success('矩阵配置已激活');
      queryClient.invalidateQueries({ queryKey: PRIORITY_MATRIX_KEYS.all });
    },
  });
}

export function useCreatePriorityRuleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: PriorityMatrixApi.createPriorityRule,
    onSuccess: () => {
      message.success('规则已创建');
      queryClient.invalidateQueries({ queryKey: PRIORITY_MATRIX_KEYS.rules() });
    },
  });
}

export default {
  useMatrixConfigsQuery,
  useActiveMatrixConfigQuery,
  useMatrixDataQuery,
  usePriorityRulesQuery,
  usePrioritySuggestionQuery,
  usePriorityDistributionQuery,
  useCalculatePriorityMutation,
  useCreateMatrixConfigMutation,
  useActivateMatrixConfigMutation,
  useCreatePriorityRuleMutation,
};

