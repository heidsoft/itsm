/**
 * 工单关联 React Query Hooks
 * 提供工单关联的数据管理和状态管理
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { TicketRelationsApi } from '@/lib/api/ticket-relations-api';
import type {
  CreateRelationRequest,
  BatchCreateRelationsRequest,
  UpdateRelationRequest,
} from '@/types/ticket-relations';

// ==================== Query Keys ====================

export const TICKET_RELATION_KEYS = {
  all: ['ticket-relations'] as const,
  lists: () => [...TICKET_RELATION_KEYS.all, 'list'] as const,
  list: (ticketId: number, filters?: any) =>
    [...TICKET_RELATION_KEYS.lists(), ticketId, filters] as const,
  details: () => [...TICKET_RELATION_KEYS.all, 'detail'] as const,
  detail: (relationId: string) =>
    [...TICKET_RELATION_KEYS.details(), relationId] as const,
  hierarchy: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'hierarchy', ticketId] as const,
  dependencies: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'dependencies', ticketId] as const,
  dependencyGraph: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'dependency-graph', ticketId] as const,
  stats: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'stats', ticketId] as const,
  graph: (ticketId: number, params?: any) =>
    [...TICKET_RELATION_KEYS.all, 'graph', ticketId, params] as const,
  suggestions: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'suggestions', ticketId] as const,
  permissions: (ticketId: number) =>
    [...TICKET_RELATION_KEYS.all, 'permissions', ticketId] as const,
};

// ==================== Query Hooks ====================

/**
 * 获取工单的所有关联
 */
export function useTicketRelationsQuery(
  ticketId: number,
  options?: {
    relationType?: string;
    direction?: string;
    includeDetails?: boolean;
    enabled?: boolean;
  }
) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.list(ticketId, options),
    queryFn: () =>
      TicketRelationsApi.getTicketRelations(ticketId, {
        relationType: options?.relationType,
        direction: options?.direction,
        includeDetails: options?.includeDetails,
      }),
    enabled: options?.enabled ?? !!ticketId,
    staleTime: 30000, // 30秒
  });
}

/**
 * 获取单个关联详情
 */
export function useRelationQuery(relationId: string, enabled: boolean = true) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.detail(relationId),
    queryFn: () => TicketRelationsApi.getRelation(relationId),
    enabled: enabled && !!relationId,
    staleTime: 30000,
  });
}

/**
 * 获取工单层级结构
 */
export function useTicketHierarchyQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.hierarchy(ticketId),
    queryFn: () => TicketRelationsApi.getHierarchy(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 30000,
  });
}

/**
 * 获取依赖列表
 */
export function useTicketDependenciesQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.dependencies(ticketId),
    queryFn: () => TicketRelationsApi.getDependencies(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 30000,
  });
}

/**
 * 获取依赖图
 */
export function useDependencyGraphQuery(
  ticketId: number,
  maxDepth?: number,
  enabled: boolean = true
) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.dependencyGraph(ticketId),
    queryFn: () => TicketRelationsApi.getDependencyGraph(ticketId, maxDepth),
    enabled: enabled && !!ticketId,
    staleTime: 60000, // 1分钟
  });
}

/**
 * 获取关联统计
 */
export function useRelationStatsQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.stats(ticketId),
    queryFn: () => TicketRelationsApi.getRelationStats(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 60000,
  });
}

/**
 * 获取关系图谱
 */
export function useRelationGraphQuery(
  ticketId: number,
  params?: {
    maxDepth?: number;
    relationTypes?: string[];
    direction?: string;
  },
  enabled: boolean = true
) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.graph(ticketId, params),
    queryFn: () => TicketRelationsApi.getRelationGraph(ticketId, params),
    enabled: enabled && !!ticketId,
    staleTime: 60000,
  });
}

/**
 * 获取关联建议
 */
export function useRelationSuggestionsQuery(
  ticketId: number,
  limit?: number,
  enabled: boolean = true
) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.suggestions(ticketId),
    queryFn: () => TicketRelationsApi.getRelationSuggestions(ticketId, limit),
    enabled: enabled && !!ticketId,
    staleTime: 300000, // 5分钟
  });
}

/**
 * 获取关联权限
 */
export function useRelationPermissionsQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: TICKET_RELATION_KEYS.permissions(ticketId),
    queryFn: () => TicketRelationsApi.getRelationPermissions(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 300000,
  });
}

// ==================== Mutation Hooks ====================

/**
 * 创建关联
 */
export function useCreateRelationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateRelationRequest) =>
      TicketRelationsApi.createRelation(request),
    onSuccess: (_, variables) => {
      message.success('关联创建成功');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.list(variables.sourceTicketId),
      });
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.list(variables.targetTicketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`创建关联失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量创建关联
 */
export function useBatchCreateRelationsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: BatchCreateRelationsRequest) =>
      TicketRelationsApi.batchCreateRelations(request),
    onSuccess: (result, variables) => {
      message.success(`成功创建 ${result.created} 个关联`);
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.list(variables.sourceTicketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`批量创建关联失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 更新关联
 */
export function useUpdateRelationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      relationId,
      request,
    }: {
      relationId: string;
      request: UpdateRelationRequest;
    }) => TicketRelationsApi.updateRelation(relationId, request),
    onSuccess: (_, variables) => {
      message.success('关联更新成功');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.detail(variables.relationId),
      });
      queryClient.invalidateQueries({ queryKey: TICKET_RELATION_KEYS.lists() });
    },
    onError: (error: any) => {
      message.error(`更新关联失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 删除关联
 */
export function useDeleteRelationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ relationId, reason }: { relationId: string; reason?: string }) =>
      TicketRelationsApi.deleteRelation(relationId, reason),
    onSuccess: () => {
      message.success('关联已删除');
      queryClient.invalidateQueries({ queryKey: TICKET_RELATION_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`删除关联失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 设置父工单
 */
export function useSetParentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      childTicketId,
      parentTicketId,
    }: {
      childTicketId: number;
      parentTicketId: number;
    }) => TicketRelationsApi.setParent(childTicketId, parentTicketId),
    onSuccess: (_, variables) => {
      message.success('父工单设置成功');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.hierarchy(variables.childTicketId),
      });
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.hierarchy(variables.parentTicketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`设置父工单失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 移除父工单
 */
export function useRemoveParentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (childTicketId: number) =>
      TicketRelationsApi.removeParent(childTicketId),
    onSuccess: (_, childTicketId) => {
      message.success('父工单已移除');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.hierarchy(childTicketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`移除父工单失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 添加依赖
 */
export function useAddDependencyMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketId: number;
      dependsOnTicketId: number;
      dependencyType: 'hard' | 'soft';
    }) => TicketRelationsApi.addDependency(data),
    onSuccess: (_, variables) => {
      message.success('依赖添加成功');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.dependencies(variables.ticketId),
      });
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.dependencyGraph(variables.ticketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`添加依赖失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 移除依赖
 */
export function useRemoveDependencyMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      ticketId,
      dependencyId,
    }: {
      ticketId: number;
      dependencyId: string;
    }) => TicketRelationsApi.removeDependency(ticketId, dependencyId),
    onSuccess: (_, variables) => {
      message.success('依赖已移除');
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.dependencies(variables.ticketId),
      });
      queryClient.invalidateQueries({
        queryKey: TICKET_RELATION_KEYS.dependencyGraph(variables.ticketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`移除依赖失败：${error.message || '未知错误'}`);
    },
  });
}

// ==================== 导出所有 Hooks ====================

export default {
  // Query Hooks
  useTicketRelationsQuery,
  useRelationQuery,
  useTicketHierarchyQuery,
  useTicketDependenciesQuery,
  useDependencyGraphQuery,
  useRelationStatsQuery,
  useRelationGraphQuery,
  useRelationSuggestionsQuery,
  useRelationPermissionsQuery,
  // Mutation Hooks
  useCreateRelationMutation,
  useBatchCreateRelationsMutation,
  useUpdateRelationMutation,
  useDeleteRelationMutation,
  useSetParentMutation,
  useRemoveParentMutation,
  useAddDependencyMutation,
  useRemoveDependencyMutation,
};

