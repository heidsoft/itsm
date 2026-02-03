/**
 * 变更分类 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { ChangeClassificationApi } from '@/lib/api/change-classification-api';
import type { ClassificationQuery } from '@/types/change-classification';

export const CHANGE_CLASSIFICATION_KEYS = {
  all: ['change-classifications'] as const,
  lists: () => [...CHANGE_CLASSIFICATION_KEYS.all, 'list'] as const,
  list: (query?: ClassificationQuery) =>
    [...CHANGE_CLASSIFICATION_KEYS.lists(), query] as const,
  details: () => [...CHANGE_CLASSIFICATION_KEYS.all, 'detail'] as const,
  detail: (id: string) =>
    [...CHANGE_CLASSIFICATION_KEYS.details(), id] as const,
  suggestion: (changeId: number) =>
    [...CHANGE_CLASSIFICATION_KEYS.all, 'suggestion', changeId] as const,
  rules: () => [...CHANGE_CLASSIFICATION_KEYS.all, 'rules'] as const,
  templates: (classificationId?: string) =>
    [...CHANGE_CLASSIFICATION_KEYS.all, 'templates', classificationId] as const,
};

// Query Hooks
export function useClassificationsQuery(query?: ClassificationQuery) {
  return useQuery({
    queryKey: CHANGE_CLASSIFICATION_KEYS.list(query),
    queryFn: () => ChangeClassificationApi.getClassifications(query),
    staleTime: 300000,
  });
}

export function useClassificationQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: CHANGE_CLASSIFICATION_KEYS.detail(id),
    queryFn: () => ChangeClassificationApi.getClassification(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useClassificationSuggestionQuery(
  changeId: number,
  enabled = true
) {
  return useQuery({
    queryKey: CHANGE_CLASSIFICATION_KEYS.suggestion(changeId),
    queryFn: () =>
      ChangeClassificationApi.getClassificationSuggestion(changeId),
    enabled: enabled && !!changeId,
    staleTime: 60000,
  });
}

export function useClassificationRulesQuery() {
  return useQuery({
    queryKey: CHANGE_CLASSIFICATION_KEYS.rules(),
    queryFn: () => ChangeClassificationApi.getClassificationRules(),
    staleTime: 300000,
  });
}

export function useChangeTemplatesQuery(classificationId?: string) {
  return useQuery({
    queryKey: CHANGE_CLASSIFICATION_KEYS.templates(classificationId),
    queryFn: () =>
      ChangeClassificationApi.getChangeTemplates({ classificationId }),
    staleTime: 300000,
  });
}

// Mutation Hooks
export function useCreateClassificationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ChangeClassificationApi.createClassification,
    onSuccess: () => {
      message.success('分类已创建');
      queryClient.invalidateQueries({
        queryKey: CHANGE_CLASSIFICATION_KEYS.lists(),
      });
    },
  });
}

export function useUpdateClassificationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof ChangeClassificationApi.updateClassification>[1];
    }) => ChangeClassificationApi.updateClassification(id, data),
    onSuccess: (_, variables) => {
      message.success('分类已更新');
      queryClient.invalidateQueries({
        queryKey: CHANGE_CLASSIFICATION_KEYS.detail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: CHANGE_CLASSIFICATION_KEYS.lists(),
      });
    },
  });
}

export function useAssessRiskMutation() {
  return useMutation({
    mutationFn: ChangeClassificationApi.assessRisk,
    onSuccess: () => {
      message.success('风险评估已完成');
    },
  });
}

export function useAnalyzeImpactMutation() {
  return useMutation({
    mutationFn: ChangeClassificationApi.analyzeImpact,
    onSuccess: () => {
      message.success('影响分析已完成');
    },
  });
}

export function useApplyClassificationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      changeId,
      classificationId,
    }: {
      changeId: number;
      classificationId: string;
    }) =>
      ChangeClassificationApi.applyClassificationSuggestion(
        changeId,
        classificationId
      ),
    onSuccess: () => {
      message.success('分类已应用');
      queryClient.invalidateQueries({ queryKey: ['changes'] });
    },
  });
}

const ChangeClassificationHooks = {
  useClassificationsQuery,
  useClassificationQuery,
  useClassificationSuggestionQuery,
  useClassificationRulesQuery,
  useChangeTemplatesQuery,
  useCreateClassificationMutation,
  useUpdateClassificationMutation,
  useAssessRiskMutation,
  useAnalyzeImpactMutation,
  useApplyClassificationMutation,
};

export default ChangeClassificationHooks;

