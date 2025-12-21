/**
 * 工作流 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { WorkflowApi } from '@/lib/api/workflow-api';
import type { WorkflowQuery, StartWorkflowRequest } from '@/types/workflow';

export const WORKFLOW_KEYS = {
  all: ['workflows'] as const,
  lists: () => [...WORKFLOW_KEYS.all, 'list'] as const,
  list: (query?: WorkflowQuery) => [...WORKFLOW_KEYS.lists(), query] as const,
  details: () => [...WORKFLOW_KEYS.all, 'detail'] as const,
  detail: (id: string) => [...WORKFLOW_KEYS.details(), id] as const,
  versions: (id: string) => [...WORKFLOW_KEYS.all, 'versions', id] as const,
  stats: (id: string) => [...WORKFLOW_KEYS.all, 'stats', id] as const,
  instances: () => [...WORKFLOW_KEYS.all, 'instances'] as const,
  instance: (id: string) => [...WORKFLOW_KEYS.instances(), id] as const,
  nodeInstances: (instanceId: string) =>
    [...WORKFLOW_KEYS.all, 'node-instances', instanceId] as const,
  templates: () => [...WORKFLOW_KEYS.all, 'templates'] as const,
};

// Query Hooks
export function useWorkflowsQuery(query?: WorkflowQuery) {
  return useQuery({
    queryKey: WORKFLOW_KEYS.list(query),
    queryFn: () => WorkflowApi.getWorkflows(query),
    staleTime: 300000,
  });
}

export function useWorkflowQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: WORKFLOW_KEYS.detail(id),
    queryFn: () => WorkflowApi.getWorkflow(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useWorkflowVersionsQuery(workflowId: string, enabled = true) {
  return useQuery({
    queryKey: WORKFLOW_KEYS.versions(workflowId),
    queryFn: () => WorkflowApi.getWorkflowVersions(workflowId),
    enabled: enabled && !!workflowId,
    staleTime: 300000,
  });
}

export function useWorkflowStatsQuery(
  workflowId: string,
  params?: { startDate?: string; endDate?: string },
  enabled = true
) {
  return useQuery({
    queryKey: [...WORKFLOW_KEYS.stats(workflowId), params],
    queryFn: () => WorkflowApi.getWorkflowStats(workflowId, params),
    enabled: enabled && !!workflowId,
    staleTime: 60000,
  });
}

export function useWorkflowInstancesQuery(params?: {
  workflowId?: string;
  ticketId?: number;
  status?: string;
  page?: number;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: [...WORKFLOW_KEYS.instances(), params],
    queryFn: () => WorkflowApi.getInstances(params),
    staleTime: 60000,
  });
}

export function useWorkflowInstanceQuery(instanceId: string, enabled = true) {
  return useQuery({
    queryKey: WORKFLOW_KEYS.instance(instanceId),
    queryFn: () => WorkflowApi.getInstance(instanceId),
    enabled: enabled && !!instanceId,
    staleTime: 30000,
    refetchInterval: 5000, // 每5秒刷新一次运行中的实例
  });
}

export function useNodeInstancesQuery(instanceId: string, enabled = true) {
  return useQuery({
    queryKey: WORKFLOW_KEYS.nodeInstances(instanceId),
    queryFn: () => WorkflowApi.getNodeInstances(instanceId),
    enabled: enabled && !!instanceId,
    staleTime: 30000,
  });
}

export function useWorkflowTemplatesQuery() {
  return useQuery({
    queryKey: WORKFLOW_KEYS.templates(),
    queryFn: () => WorkflowApi.getTemplates(),
    staleTime: 300000,
  });
}

// Mutation Hooks
export function useCreateWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: WorkflowApi.createWorkflow,
    onSuccess: () => {
      message.success('工作流已创建');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useUpdateWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof WorkflowApi.updateWorkflow>[1];
    }) => WorkflowApi.updateWorkflow(id, data),
    onSuccess: (_, variables) => {
      message.success('工作流已更新');
      queryClient.invalidateQueries({
        queryKey: WORKFLOW_KEYS.detail(variables.id),
      });
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useDeleteWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => WorkflowApi.deleteWorkflow(id),
    onSuccess: () => {
      message.success('工作流已删除');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useActivateWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => WorkflowApi.activateWorkflow(id),
    onSuccess: (_, id) => {
      message.success('工作流已激活');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.detail(id) });
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useDeactivateWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => WorkflowApi.deactivateWorkflow(id),
    onSuccess: (_, id) => {
      message.success('工作流已停用');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.detail(id) });
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useCloneWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      WorkflowApi.cloneWorkflow(id, name),
    onSuccess: () => {
      message.success('工作流已复制');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.lists() });
    },
  });
}

export function useStartWorkflowMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: StartWorkflowRequest) =>
      WorkflowApi.startWorkflow(request),
    onSuccess: () => {
      message.success('工作流已启动');
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.instances() });
    },
  });
}

export function useCancelInstanceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ instanceId, reason }: { instanceId: string; reason?: string }) =>
      WorkflowApi.cancelInstance(instanceId, reason),
    onSuccess: (_, variables) => {
      message.success('工作流实例已取消');
      queryClient.invalidateQueries({
        queryKey: WORKFLOW_KEYS.instance(variables.instanceId),
      });
      queryClient.invalidateQueries({ queryKey: WORKFLOW_KEYS.instances() });
    },
  });
}

export function useCompleteNodeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: WorkflowApi.completeNode,
    onSuccess: (_, variables) => {
      message.success('节点任务已完成');
      queryClient.invalidateQueries({
        queryKey: WORKFLOW_KEYS.instance(variables.instanceId),
      });
      queryClient.invalidateQueries({
        queryKey: WORKFLOW_KEYS.nodeInstances(variables.instanceId),
      });
    },
  });
}

export default {
  useWorkflowsQuery,
  useWorkflowQuery,
  useWorkflowVersionsQuery,
  useWorkflowStatsQuery,
  useWorkflowInstancesQuery,
  useWorkflowInstanceQuery,
  useNodeInstancesQuery,
  useWorkflowTemplatesQuery,
  useCreateWorkflowMutation,
  useUpdateWorkflowMutation,
  useDeleteWorkflowMutation,
  useActivateWorkflowMutation,
  useDeactivateWorkflowMutation,
  useCloneWorkflowMutation,
  useStartWorkflowMutation,
  useCancelInstanceMutation,
  useCompleteNodeMutation,
};

