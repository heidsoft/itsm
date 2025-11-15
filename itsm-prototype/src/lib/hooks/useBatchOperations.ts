/**
 * 批量操作 React Query Hooks
 * 提供批量操作的数据管理和状态管理
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { BatchOperationsApi } from '@/lib/api/batch-operations-api';
import type {
  BatchOperationRequest,
  BatchOperationResponse,
  BatchOperationProgress,
  BatchOperationLog,
  BatchOperationStats,
  BatchOperationPermissions,
  BatchExportConfig,
} from '@/types/batch-operations';

// ==================== Query Keys ====================

export const BATCH_OPERATION_KEYS = {
  all: ['batch-operations'] as const,
  progress: (operationId: string) =>
    [...BATCH_OPERATION_KEYS.all, 'progress', operationId] as const,
  logs: (filters?: any) =>
    [...BATCH_OPERATION_KEYS.all, 'logs', filters] as const,
  stats: (filters?: any) =>
    [...BATCH_OPERATION_KEYS.all, 'stats', filters] as const,
  permissions: () => [...BATCH_OPERATION_KEYS.all, 'permissions'] as const,
  scheduled: () => [...BATCH_OPERATION_KEYS.all, 'scheduled'] as const,
  exportStatus: (exportId: string) =>
    [...BATCH_OPERATION_KEYS.all, 'export', exportId] as const,
};

// ==================== Mutation Hooks ====================

/**
 * 批量分配工单
 */
export function useBatchAssignMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      assigneeId?: number;
      teamId?: number;
      assignmentRule?: 'round_robin' | 'load_balance' | 'manual';
      comment?: string;
    }) => BatchOperationsApi.batchAssignTickets(data),
    onSuccess: (response) => {
      message.success(
        `批量分配成功：${response.successCount}个工单已分配`
      );
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量分配失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量更新状态
 */
export function useBatchUpdateStatusMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      status: string;
      resolution?: string;
      comment?: string;
    }) => BatchOperationsApi.batchUpdateStatus(data),
    onSuccess: (response) => {
      message.success(
        `批量更新成功：${response.successCount}个工单已更新`
      );
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量更新失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量更新优先级
 */
export function useBatchUpdatePriorityMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      priority: string;
      comment?: string;
    }) => BatchOperationsApi.batchUpdatePriority(data),
    onSuccess: (response) => {
      message.success(
        `批量更新优先级成功：${response.successCount}个工单`
      );
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量更新优先级失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量更新字段
 */
export function useBatchUpdateFieldsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      customFields: Record<string, any>;
      comment?: string;
    }) => BatchOperationsApi.batchUpdateFields(data),
    onSuccess: (response) => {
      message.success(
        `批量更新字段成功：${response.successCount}个工单`
      );
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量更新字段失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量添加标签
 */
export function useBatchAddTagsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      tags: string[];
      comment?: string;
    }) => BatchOperationsApi.batchAddTags(data),
    onSuccess: (response) => {
      message.success(`批量添加标签成功：${response.successCount}个工单`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量添加标签失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量删除标签
 */
export function useBatchRemoveTagsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      tags: string[];
      comment?: string;
    }) => BatchOperationsApi.batchRemoveTags(data),
    onSuccess: (response) => {
      message.success(`批量删除标签成功：${response.successCount}个工单`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量删除标签失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量删除工单
 */
export function useBatchDeleteMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      reason?: string;
      hardDelete?: boolean;
    }) => BatchOperationsApi.batchDeleteTickets(data),
    onSuccess: (response) => {
      message.success(`批量删除成功：${response.successCount}个工单`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量删除失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量关闭工单
 */
export function useBatchCloseMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      closureReason?: string;
      resolution?: string;
      comment?: string;
    }) => BatchOperationsApi.batchCloseTickets(data),
    onSuccess: (response) => {
      message.success(`批量关闭成功：${response.successCount}个工单`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量关闭失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量重新打开工单
 */
export function useBatchReopenMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      ticketIds: number[];
      reason?: string;
      comment?: string;
    }) => BatchOperationsApi.batchReopenTickets(data),
    onSuccess: (response) => {
      message.success(`批量重新打开成功：${response.successCount}个工单`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量重新打开失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 批量导出工单
 */
export function useBatchExportMutation() {
  return useMutation({
    mutationFn: (data: {
      ticketIds?: number[];
      filters?: Record<string, any>;
      config: BatchExportConfig;
    }) => BatchOperationsApi.batchExportTickets(data),
    onSuccess: (blob, variables) => {
      // 创建下载链接
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = variables.config.fileName || `tickets_export.${variables.config.format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('导出成功！');
    },
    onError: (error: any) => {
      message.error(`导出失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 撤销批量操作
 */
export function useUndoBatchOperationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (operationId: string) =>
      BatchOperationsApi.undoBatchOperation(operationId),
    onSuccess: (response) => {
      message.success(`撤销成功：${response.successCount}个工单已恢复`);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`撤销失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 执行通用批量操作
 */
export function useBatchOperationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: BatchOperationRequest) =>
      BatchOperationsApi.executeBatchOperation(request),
    onSuccess: (response) => {
      message.success(
        `批量操作成功：${response.successCount}个工单，失败${response.failedCount}个`
      );
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      queryClient.invalidateQueries({ queryKey: BATCH_OPERATION_KEYS.all });
    },
    onError: (error: any) => {
      message.error(`批量操作失败：${error.message || '未知错误'}`);
    },
  });
}

// ==================== Query Hooks ====================

/**
 * 获取批量操作进度（支持实时轮询）
 */
export function useBatchOperationProgressQuery(
  operationId: string,
  options?: {
    enabled?: boolean;
    refetchInterval?: number;
  }
) {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.progress(operationId),
    queryFn: () => BatchOperationsApi.getBatchOperationProgress(operationId),
    enabled: options?.enabled ?? !!operationId,
    refetchInterval: options?.refetchInterval ?? 2000, // 每2秒刷新一次
    staleTime: 1000,
  });
}

/**
 * 获取批量操作日志列表
 */
export function useBatchOperationLogsQuery(params?: {
  page?: number;
  pageSize?: number;
  operationType?: string;
  status?: string;
}) {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.logs(params),
    queryFn: () => BatchOperationsApi.getBatchOperationLogs(params),
    staleTime: 30000, // 30秒
  });
}

/**
 * 获取批量操作统计
 */
export function useBatchOperationStatsQuery(params?: {
  startDate?: string;
  endDate?: string;
  groupBy?: 'day' | 'week' | 'month';
}) {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.stats(params),
    queryFn: () => BatchOperationsApi.getBatchOperationStats(params),
    staleTime: 60000, // 1分钟
  });
}

/**
 * 获取批量操作权限
 */
export function useBatchOperationPermissionsQuery() {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.permissions(),
    queryFn: () => BatchOperationsApi.getBatchOperationPermissions(),
    staleTime: 300000, // 5分钟
  });
}

/**
 * 获取调度的批量操作列表
 */
export function useScheduledOperationsQuery() {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.scheduled(),
    queryFn: () => BatchOperationsApi.getScheduledOperations(),
    staleTime: 30000, // 30秒
  });
}

/**
 * 获取我的批量操作统计
 */
export function useMyBatchOperationStatsQuery() {
  return useQuery({
    queryKey: [...BATCH_OPERATION_KEYS.all, 'my-stats'],
    queryFn: () => BatchOperationsApi.getMyBatchOperationStats(),
    staleTime: 60000, // 1分钟
  });
}

/**
 * 获取导出状态
 */
export function useExportStatusQuery(exportId: string, enabled: boolean = true) {
  return useQuery({
    queryKey: BATCH_OPERATION_KEYS.exportStatus(exportId),
    queryFn: () => BatchOperationsApi.getExportStatus(exportId),
    enabled: enabled && !!exportId,
    refetchInterval: 3000, // 每3秒刷新
    staleTime: 1000,
  });
}

// ==================== 导出所有 Hooks ====================

export default {
  // Mutations
  useBatchAssignMutation,
  useBatchUpdateStatusMutation,
  useBatchUpdatePriorityMutation,
  useBatchUpdateFieldsMutation,
  useBatchAddTagsMutation,
  useBatchRemoveTagsMutation,
  useBatchDeleteMutation,
  useBatchCloseMutation,
  useBatchReopenMutation,
  useBatchExportMutation,
  useUndoBatchOperationMutation,
  useBatchOperationMutation,
  // Queries
  useBatchOperationProgressQuery,
  useBatchOperationLogsQuery,
  useBatchOperationStatsQuery,
  useBatchOperationPermissionsQuery,
  useScheduledOperationsQuery,
  useMyBatchOperationStatsQuery,
  useExportStatusQuery,
};

