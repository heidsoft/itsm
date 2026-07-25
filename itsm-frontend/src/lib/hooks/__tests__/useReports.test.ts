import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useReportsQuery,
  useReportQuery,
  useExecutionHistoryQuery,
  useExecutionResultQuery,
  useTemplatesQuery as useReportTemplatesQuery,
  useTemplateQuery as useReportTemplateQuery,
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
  REPORTS_KEYS,
} from '../useReports';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock ReportsApi
jest.mock('@/lib/api/reports-api', () => ({
  ReportsApi: {
    getReports: jest.fn(),
    getReport: jest.fn(),
    getExecutionHistory: jest.fn(),
    getExecutionResult: jest.fn(),
    getTemplates: jest.fn(),
    getTemplate: jest.fn(),
    getDatasets: jest.fn(),
    getDataset: jest.fn(),
    getStats: jest.fn(),
    getPerformance: jest.fn(),
    getRecentReports: jest.fn(),
    getFavoriteReports: jest.fn(),
    createReport: jest.fn(),
    updateReport: jest.fn(),
    deleteReport: jest.fn(),
    cloneReport: jest.fn(),
    executeReport: jest.fn(),
    exportReport: jest.fn(),
    emailReport: jest.fn(),
    createSchedule: jest.fn(),
    updateSchedule: jest.fn(),
    deleteSchedule: jest.fn(),
    createFromTemplate: jest.fn(),
    saveAsTemplate: jest.fn(),
    favoriteReport: jest.fn(),
    unfavoriteReport: jest.fn(),
    shareReport: jest.fn(),
    previewData: jest.fn(),
    validateQuery: jest.fn(),
  },
}));

import { ReportsApi } from '@/lib/api/reports-api';
const mockApi = ReportsApi as jest.Mocked<typeof ReportsApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useReports hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('REPORTS_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(REPORTS_KEYS.all).toEqual(['reports']);
      expect(REPORTS_KEYS.lists()).toEqual(['reports', 'list']);
      expect(REPORTS_KEYS.detail('r1')).toEqual(['reports', 'detail', 'r1']);
      expect(REPORTS_KEYS.executions('r1')).toEqual(['reports', 'executions', 'r1']);
      expect(REPORTS_KEYS.executionDetail('e1')).toEqual(['reports', 'execution', 'e1']);
      expect(REPORTS_KEYS.templates()).toEqual(['reports', 'templates']);
      expect(REPORTS_KEYS.datasets()).toEqual(['reports', 'datasets']);
      expect(REPORTS_KEYS.stats()).toEqual(['reports', 'stats']);
      expect(REPORTS_KEYS.performance('r1')).toEqual(['reports', 'performance', 'r1']);
      expect(REPORTS_KEYS.recent()).toEqual(['reports', 'recent']);
      expect(REPORTS_KEYS.favorites()).toEqual(['reports', 'favorites']);
    });
  });

  describe('useReportsQuery', () => {
    it('should fetch reports', async () => {
      mockApi.getReports.mockResolvedValue({ items: [], total: 0 } as any);

      const { result } = renderHook(() => useReportsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getReports).toHaveBeenCalledWith(undefined);
    });

    it('should pass query params', async () => {
      mockApi.getReports.mockResolvedValue({ items: [], total: 0 } as any);
      const query = { category: 'tickets' } as any;

      const { result } = renderHook(() => useReportsQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getReports).toHaveBeenCalledWith(query);
    });
  });

  describe('useReportQuery', () => {
    it('should fetch a single report', async () => {
      mockApi.getReport.mockResolvedValue({ id: 'r1', name: 'Report 1' } as any);

      const { result } = renderHook(() => useReportQuery('r1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getReport).toHaveBeenCalledWith('r1');
    });

    it('should not fetch when id is empty', () => {
      renderHook(() => useReportQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getReport).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useReportQuery('r1', false), { wrapper: createWrapper() });
      expect(mockApi.getReport).not.toHaveBeenCalled();
    });
  });

  describe('useExecutionHistoryQuery', () => {
    it('should fetch execution history', async () => {
      mockApi.getExecutionHistory.mockResolvedValue({ items: [], total: 0 } as any);

      const { result } = renderHook(() => useExecutionHistoryQuery('r1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getExecutionHistory).toHaveBeenCalledWith('r1', undefined);
    });

    it('should not fetch when reportId is empty', () => {
      renderHook(() => useExecutionHistoryQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getExecutionHistory).not.toHaveBeenCalled();
    });
  });

  describe('useExecutionResultQuery', () => {
    it('should fetch execution result', async () => {
      mockApi.getExecutionResult.mockResolvedValue({ data: [] } as any);

      const { result } = renderHook(() => useExecutionResultQuery('e1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getExecutionResult).toHaveBeenCalledWith('e1');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useExecutionResultQuery('e1', false), { wrapper: createWrapper() });
      expect(mockApi.getExecutionResult).not.toHaveBeenCalled();
    });
  });

  describe('useDatasetsQuery', () => {
    it('should fetch datasets', async () => {
      mockApi.getDatasets.mockResolvedValue([]);

      const { result } = renderHook(() => useDatasetsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getDatasets).toHaveBeenCalled();
    });
  });

  describe('useDatasetQuery', () => {
    it('should fetch a single dataset', async () => {
      mockApi.getDataset.mockResolvedValue({ id: 'd1' } as any);

      const { result } = renderHook(() => useDatasetQuery('d1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getDataset).toHaveBeenCalledWith('d1');
    });

    it('should not fetch when id is empty', () => {
      renderHook(() => useDatasetQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getDataset).not.toHaveBeenCalled();
    });
  });

  describe('useReportStatsQuery', () => {
    it('should fetch report stats', async () => {
      mockApi.getStats.mockResolvedValue({ totalReports: 10 } as any);

      const { result } = renderHook(() => useReportStatsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getStats).toHaveBeenCalled();
    });
  });

  describe('useReportPerformanceQuery', () => {
    it('should fetch report performance', async () => {
      mockApi.getPerformance.mockResolvedValue({ avgTime: 500 } as any);

      const { result } = renderHook(() => useReportPerformanceQuery('r1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getPerformance).toHaveBeenCalledWith('r1', undefined);
    });

    it('should not fetch when reportId is empty', () => {
      renderHook(() => useReportPerformanceQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getPerformance).not.toHaveBeenCalled();
    });
  });

  describe('useRecentReportsQuery', () => {
    it('should fetch recent reports with default limit', async () => {
      mockApi.getRecentReports.mockResolvedValue([]);

      const { result } = renderHook(() => useRecentReportsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRecentReports).toHaveBeenCalledWith(10);
    });

    it('should pass custom limit', async () => {
      mockApi.getRecentReports.mockResolvedValue([]);

      const { result } = renderHook(() => useRecentReportsQuery(5), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRecentReports).toHaveBeenCalledWith(5);
    });
  });

  describe('useFavoriteReportsQuery', () => {
    it('should fetch favorite reports', async () => {
      mockApi.getFavoriteReports.mockResolvedValue([]);

      const { result } = renderHook(() => useFavoriteReportsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getFavoriteReports).toHaveBeenCalled();
    });
  });

  describe('useCreateReportMutation', () => {
    it('should create a report', async () => {
      mockApi.createReport.mockResolvedValue({ id: 'new-r' } as any);

      const { result } = renderHook(() => useCreateReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Report' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createReport).toHaveBeenCalled();
    });
  });

  describe('useUpdateReportMutation', () => {
    it('should update a report', async () => {
      mockApi.updateReport.mockResolvedValue({ id: 'r1' } as any);

      const { result } = renderHook(() => useUpdateReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 'r1', data: { name: 'Updated' } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateReport).toHaveBeenCalledWith('r1', { name: 'Updated' });
    });
  });

  describe('useDeleteReportMutation', () => {
    it('should delete a report', async () => {
      mockApi.deleteReport.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('r1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteReport).toHaveBeenCalledWith('r1');
    });
  });

  describe('useCloneReportMutation', () => {
    it('should clone a report', async () => {
      mockApi.cloneReport.mockResolvedValue({ id: 'r2' } as any);

      const { result } = renderHook(() => useCloneReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 'r1', name: 'Clone' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.cloneReport).toHaveBeenCalledWith('r1', 'Clone');
    });
  });

  describe('useExecuteReportMutation', () => {
    it('should execute a report', async () => {
      mockApi.executeReport.mockResolvedValue({ executionId: 'e1' } as any);

      const { result } = renderHook(() => useExecuteReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });

    it('should handle execution error', async () => {
      mockApi.executeReport.mockRejectedValue(new Error('Execution failed'));

      const { result } = renderHook(() => useExecuteReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1' } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useExportReportMutation', () => {
    it('should export a report', async () => {
      const mockBlob = new Blob(['data']);
      mockApi.exportReport.mockResolvedValue(mockBlob);
      // Mock URL methods
      global.URL.createObjectURL = jest.fn(() => 'blob:url');
      global.URL.revokeObjectURL = jest.fn();

      const { result } = renderHook(() => useExportReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1', format: 'pdf' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useEmailReportMutation', () => {
    it('should email a report', async () => {
      mockApi.emailReport.mockResolvedValue(undefined);

      const { result } = renderHook(() => useEmailReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1', email: 'test@example.com' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });

    it('should handle email error', async () => {
      mockApi.emailReport.mockRejectedValue(new Error('Send failed'));

      const { result } = renderHook(() => useEmailReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1', email: 'test@example.com' } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useCreateScheduleMutation', () => {
    it('should create a schedule', async () => {
      mockApi.createSchedule.mockResolvedValue({ id: 's1' } as any);

      const { result } = renderHook(() => useCreateScheduleMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1', schedule: { cron: '0 0 * * *' } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createSchedule).toHaveBeenCalledWith('r1', { cron: '0 0 * * *' });
    });
  });

  describe('useDeleteScheduleMutation', () => {
    it('should delete a schedule', async () => {
      mockApi.deleteSchedule.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteScheduleMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('r1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteSchedule).toHaveBeenCalledWith('r1');
    });
  });

  describe('useCreateFromTemplateMutation', () => {
    it('should create from template', async () => {
      mockApi.createFromTemplate.mockResolvedValue({ id: 'new-r' } as any);

      const { result } = renderHook(() => useCreateFromTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateId: 't1', name: 'From Template' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createFromTemplate).toHaveBeenCalledWith('t1', 'From Template');
    });
  });

  describe('useFavoriteReportMutation', () => {
    it('should favorite a report', async () => {
      mockApi.favoriteReport.mockResolvedValue(undefined);

      const { result } = renderHook(() => useFavoriteReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('r1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.favoriteReport).toHaveBeenCalledWith('r1');
    });
  });

  describe('useUnfavoriteReportMutation', () => {
    it('should unfavorite a report', async () => {
      mockApi.unfavoriteReport.mockResolvedValue(undefined);

      const { result } = renderHook(() => useUnfavoriteReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('r1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.unfavoriteReport).toHaveBeenCalledWith('r1');
    });
  });

  describe('useShareReportMutation', () => {
    it('should share a report', async () => {
      mockApi.shareReport.mockResolvedValue(undefined);

      const { result } = renderHook(() => useShareReportMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ reportId: 'r1', params: { users: [1, 2] } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.shareReport).toHaveBeenCalledWith('r1', { users: [1, 2] });
    });
  });

  describe('usePreviewDataMutation', () => {
    it('should preview data', async () => {
      mockApi.previewData.mockResolvedValue({ data: [] } as any);

      const { result } = renderHook(() => usePreviewDataMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ query: 'SELECT *' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });

    it('should handle preview error', async () => {
      mockApi.previewData.mockRejectedValue(new Error('Preview failed'));

      const { result } = renderHook(() => usePreviewDataMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ query: 'BAD QUERY' } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useValidateQueryMutation', () => {
    it('should validate a query', async () => {
      mockApi.validateQuery.mockResolvedValue({ valid: true });

      const { result } = renderHook(() => useValidateQueryMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ query: 'SELECT *' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });
});
