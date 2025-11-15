/**
 * 报表系统 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { ReportsApi } from '@/lib/api/reports-api';
import type { ReportQuery } from '@/types/reports';

export const REPORTS_KEYS = {
  all: ['reports'] as const,
  lists: () => [...REPORTS_KEYS.all, 'list'] as const,
  list: (query?: ReportQuery) => [...REPORTS_KEYS.lists(), query] as const,
  details: () => [...REPORTS_KEYS.all, 'detail'] as const,
  detail: (id: string) => [...REPORTS_KEYS.details(), id] as const,
  executions: (reportId: string) =>
    [...REPORTS_KEYS.all, 'executions', reportId] as const,
  executionDetail: (executionId: string) =>
    [...REPORTS_KEYS.all, 'execution', executionId] as const,
  templates: () => [...REPORTS_KEYS.all, 'templates'] as const,
  datasets: () => [...REPORTS_KEYS.all, 'datasets'] as const,
  stats: () => [...REPORTS_KEYS.all, 'stats'] as const,
  performance: (reportId: string) =>
    [...REPORTS_KEYS.all, 'performance', reportId] as const,
  recent: () => [...REPORTS_KEYS.all, 'recent'] as const,
  favorites: () => [...REPORTS_KEYS.all, 'favorites'] as const,
};

// ==================== Query Hooks ====================

export function useReportsQuery(query?: ReportQuery) {
  return useQuery({
    queryKey: REPORTS_KEYS.list(query),
    queryFn: () => ReportsApi.getReports(query),
    staleTime: 60000,
  });
}

export function useReportQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: REPORTS_KEYS.detail(id),
    queryFn: () => ReportsApi.getReport(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useExecutionHistoryQuery(
  reportId: string,
  params?: {
    page?: number;
    pageSize?: number;
    startDate?: string;
    endDate?: string;
  },
  enabled = true
) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.executions(reportId), params],
    queryFn: () => ReportsApi.getExecutionHistory(reportId, params),
    enabled: enabled && !!reportId,
    staleTime: 60000,
  });
}

export function useExecutionResultQuery(executionId: string, enabled = true) {
  return useQuery({
    queryKey: REPORTS_KEYS.executionDetail(executionId),
    queryFn: () => ReportsApi.getExecutionResult(executionId),
    enabled: enabled && !!executionId,
    staleTime: 300000,
  });
}

export function useTemplatesQuery(params?: {
  category?: string;
  tags?: string[];
  page?: number;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.templates(), params],
    queryFn: () => ReportsApi.getTemplates(params),
    staleTime: 300000,
  });
}

export function useTemplateQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.templates(), id],
    queryFn: () => ReportsApi.getTemplate(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useDatasetsQuery() {
  return useQuery({
    queryKey: REPORTS_KEYS.datasets(),
    queryFn: () => ReportsApi.getDatasets(),
    staleTime: 600000,
  });
}

export function useDatasetQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.datasets(), id],
    queryFn: () => ReportsApi.getDataset(id),
    enabled: enabled && !!id,
    staleTime: 600000,
  });
}

export function useReportStatsQuery() {
  return useQuery({
    queryKey: REPORTS_KEYS.stats(),
    queryFn: () => ReportsApi.getStats(),
    staleTime: 300000,
  });
}

export function useReportPerformanceQuery(
  reportId: string,
  params?: {
    startDate?: string;
    endDate?: string;
  },
  enabled = true
) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.performance(reportId), params],
    queryFn: () => ReportsApi.getPerformance(reportId, params),
    enabled: enabled && !!reportId,
    staleTime: 300000,
  });
}

export function useRecentReportsQuery(limit = 10) {
  return useQuery({
    queryKey: [...REPORTS_KEYS.recent(), limit],
    queryFn: () => ReportsApi.getRecentReports(limit),
    staleTime: 60000,
  });
}

export function useFavoriteReportsQuery() {
  return useQuery({
    queryKey: REPORTS_KEYS.favorites(),
    queryFn: () => ReportsApi.getFavoriteReports(),
    staleTime: 300000,
  });
}

// ==================== Mutation Hooks ====================

export function useCreateReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ReportsApi.createReport,
    onSuccess: () => {
      message.success('报表已创建');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.stats() });
    },
  });
}

export function useUpdateReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof ReportsApi.updateReport>[1];
    }) => ReportsApi.updateReport(id, data),
    onSuccess: (_, variables) => {
      message.success('报表已更新');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.detail(variables.id),
      });
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.lists() });
    },
  });
}

export function useDeleteReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => ReportsApi.deleteReport(id),
    onSuccess: () => {
      message.success('报表已删除');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.stats() });
    },
  });
}

export function useCloneReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      ReportsApi.cloneReport(id, name),
    onSuccess: () => {
      message.success('报表已复制');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.lists() });
    },
  });
}

export function useExecuteReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ReportsApi.executeReport,
    onSuccess: (_, variables) => {
      message.success('报表执行成功');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.executions(variables.reportId),
      });
    },
    onError: () => {
      message.error('报表执行失败');
    },
  });
}

export function useExportReportMutation() {
  return useMutation({
    mutationFn: ReportsApi.exportReport,
    onSuccess: (blob, variables) => {
      // 下载文件
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `report.${variables.format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('报表已导出');
    },
    onError: () => {
      message.error('报表导出失败');
    },
  });
}

export function useEmailReportMutation() {
  return useMutation({
    mutationFn: ReportsApi.emailReport,
    onSuccess: () => {
      message.success('报表已发送');
    },
    onError: () => {
      message.error('报表发送失败');
    },
  });
}

export function useCreateScheduleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      reportId,
      schedule,
    }: {
      reportId: string;
      schedule: Parameters<typeof ReportsApi.createSchedule>[1];
    }) => ReportsApi.createSchedule(reportId, schedule),
    onSuccess: (_, variables) => {
      message.success('调度已创建');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.detail(variables.reportId),
      });
    },
  });
}

export function useUpdateScheduleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      reportId,
      schedule,
    }: {
      reportId: string;
      schedule: Parameters<typeof ReportsApi.updateSchedule>[1];
    }) => ReportsApi.updateSchedule(reportId, schedule),
    onSuccess: (_, variables) => {
      message.success('调度已更新');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.detail(variables.reportId),
      });
    },
  });
}

export function useDeleteScheduleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reportId: string) => ReportsApi.deleteSchedule(reportId),
    onSuccess: (_, reportId) => {
      message.success('调度已删除');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.detail(reportId),
      });
    },
  });
}

export function useCreateFromTemplateMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ templateId, name }: { templateId: string; name: string }) =>
      ReportsApi.createFromTemplate(templateId, name),
    onSuccess: () => {
      message.success('已从模板创建报表');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.lists() });
    },
  });
}

export function useSaveAsTemplateMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      reportId,
      params,
    }: {
      reportId: string;
      params: Parameters<typeof ReportsApi.saveAsTemplate>[1];
    }) => ReportsApi.saveAsTemplate(reportId, params),
    onSuccess: () => {
      message.success('已保存为模板');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.templates() });
    },
  });
}

export function useFavoriteReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reportId: string) => ReportsApi.favoriteReport(reportId),
    onSuccess: () => {
      message.success('已添加到收藏');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.favorites() });
    },
  });
}

export function useUnfavoriteReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reportId: string) => ReportsApi.unfavoriteReport(reportId),
    onSuccess: () => {
      message.success('已取消收藏');
      queryClient.invalidateQueries({ queryKey: REPORTS_KEYS.favorites() });
    },
  });
}

export function useShareReportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      reportId,
      params,
    }: {
      reportId: string;
      params: Parameters<typeof ReportsApi.shareReport>[1];
    }) => ReportsApi.shareReport(reportId, params),
    onSuccess: (_, variables) => {
      message.success('报表已分享');
      queryClient.invalidateQueries({
        queryKey: REPORTS_KEYS.detail(variables.reportId),
      });
    },
  });
}

export function usePreviewDataMutation() {
  return useMutation({
    mutationFn: ReportsApi.previewData,
    onError: () => {
      message.error('数据预览失败');
    },
  });
}

export function useValidateQueryMutation() {
  return useMutation({
    mutationFn: ReportsApi.validateQuery,
  });
}

export default {
  useReportsQuery,
  useReportQuery,
  useExecutionHistoryQuery,
  useExecutionResultQuery,
  useTemplatesQuery,
  useTemplateQuery,
  useDatasetsQuery,
  useDatasetQuery,
  useReportStatsQuery,
  useReportPerformanceQuery,
  useRecentReportsQuery,
  useFavoriteReportsQuery,
  useCreateReportMutation,
  useUpdateReportMutation,
  useDeleteReportMutation,
  useCloneReportMutation,
  useExecuteReportMutation,
  useExportReportMutation,
  useEmailReportMutation,
  useCreateScheduleMutation,
  useUpdateScheduleMutation,
  useDeleteScheduleMutation,
  useCreateFromTemplateMutation,
  useSaveAsTemplateMutation,
  useFavoriteReportMutation,
  useUnfavoriteReportMutation,
  useShareReportMutation,
  usePreviewDataMutation,
  useValidateQueryMutation,
};

