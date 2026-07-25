import { renderHook, waitFor, act } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
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
  useBatchOperationProgressQuery,
  useBatchOperationLogsQuery,
  useBatchOperationStatsQuery,
  useBatchOperationPermissionsQuery,
  useScheduledOperationsQuery,
  useMyBatchOperationStatsQuery,
  useExportStatusQuery,
  BATCH_OPERATION_KEYS,
} from '../useBatchOperations';

jest.mock('antd', () => ({
  message: { success: jest.fn(), error: jest.fn() },
}));

jest.mock('@/lib/api/batch-operations-api', () => ({
  BatchOperationsApi: {
    batchAssignTickets: jest.fn(),
    batchUpdateStatus: jest.fn(),
    batchUpdatePriority: jest.fn(),
    batchUpdateFields: jest.fn(),
    batchAddTags: jest.fn(),
    batchRemoveTags: jest.fn(),
    batchDeleteTickets: jest.fn(),
    batchCloseTickets: jest.fn(),
    batchReopenTickets: jest.fn(),
    batchExportTickets: jest.fn(),
    undoBatchOperation: jest.fn(),
    executeBatchOperation: jest.fn(),
    getBatchOperationProgress: jest.fn(),
    getBatchOperationLogs: jest.fn(),
    getBatchOperationStats: jest.fn(),
    getBatchOperationPermissions: jest.fn(),
    getScheduledOperations: jest.fn(),
    getMyBatchOperationStats: jest.fn(),
    getExportStatus: jest.fn(),
  },
}));

import { BatchOperationsApi } from '@/lib/api/batch-operations-api';
import { message } from 'antd';
const mockApi = BatchOperationsApi as jest.Mocked<typeof BatchOperationsApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useBatchOperations hooks', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('BATCH_OPERATION_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(BATCH_OPERATION_KEYS.all).toEqual(['batch-operations']);
      expect(BATCH_OPERATION_KEYS.progress('op-1')).toEqual(['batch-operations', 'progress', 'op-1']);
      expect(BATCH_OPERATION_KEYS.logs({ page: 1 })).toEqual(['batch-operations', 'logs', { page: 1 }]);
      expect(BATCH_OPERATION_KEYS.stats()).toEqual(['batch-operations', 'stats', undefined]);
      expect(BATCH_OPERATION_KEYS.permissions()).toEqual(['batch-operations', 'permissions']);
      expect(BATCH_OPERATION_KEYS.scheduled()).toEqual(['batch-operations', 'scheduled']);
      expect(BATCH_OPERATION_KEYS.exportStatus('exp-1')).toEqual(['batch-operations', 'export', 'exp-1']);
    });
  });

  describe('useBatchAssignMutation', () => {
    it('should batch assign tickets on success', async () => {
      mockApi.batchAssignTickets.mockResolvedValue({ successCount: 3 } as any);
      const { result } = renderHook(() => useBatchAssignMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2, 3], assigneeId: 10 }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.batchAssignTickets).toHaveBeenCalledWith({ ticketIds: [1, 2, 3], assigneeId: 10 });
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error with Error instance', async () => {
      mockApi.batchAssignTickets.mockRejectedValue(new Error('Network error'));
      const { result } = renderHook(() => useBatchAssignMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], assigneeId: 1 }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalledWith(expect.stringContaining('Network error'));
    });

    it('should handle error with non-Error value', async () => {
      mockApi.batchAssignTickets.mockRejectedValue('string error');
      const { result } = renderHook(() => useBatchAssignMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], assigneeId: 1 }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalledWith(expect.stringContaining('未知错误'));
    });
  });

  describe('useBatchUpdateStatusMutation', () => {
    it('should update status on success', async () => {
      mockApi.batchUpdateStatus.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useBatchUpdateStatusMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], status: 'closed' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchUpdateStatus.mockRejectedValue(new Error('fail'));
      const { result } = renderHook(() => useBatchUpdateStatusMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], status: 'closed' }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalled();
    });
  });

  describe('useBatchUpdatePriorityMutation', () => {
    it('should update priority on success', async () => {
      mockApi.batchUpdatePriority.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useBatchUpdatePriorityMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], priority: 'high' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchUpdatePriority.mockRejectedValue(new Error('prio error'));
      const { result } = renderHook(() => useBatchUpdatePriorityMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], priority: 'high' }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalledWith(expect.stringContaining('prio error'));
    });
  });

  describe('useBatchUpdateFieldsMutation', () => {
    it('should update fields on success', async () => {
      mockApi.batchUpdateFields.mockResolvedValue({ successCount: 1 } as any);
      const { result } = renderHook(() => useBatchUpdateFieldsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], customFields: { foo: 'bar' } }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchUpdateFields.mockRejectedValue(new Error('fields error'));
      const { result } = renderHook(() => useBatchUpdateFieldsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], customFields: {} }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchAddTagsMutation', () => {
    it('should add tags on success', async () => {
      mockApi.batchAddTags.mockResolvedValue({ successCount: 3 } as any);
      const { result } = renderHook(() => useBatchAddTagsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], tags: ['urgent'] }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchAddTags.mockRejectedValue(new Error('tag error'));
      const { result } = renderHook(() => useBatchAddTagsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], tags: ['a'] }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchRemoveTagsMutation', () => {
    it('should remove tags on success', async () => {
      mockApi.batchRemoveTags.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useBatchRemoveTagsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], tags: ['old'] }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchRemoveTags.mockRejectedValue('non-error');
      const { result } = renderHook(() => useBatchRemoveTagsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1], tags: ['x'] }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalledWith(expect.stringContaining('未知错误'));
    });
  });

  describe('useBatchDeleteMutation', () => {
    it('should delete tickets on success', async () => {
      mockApi.batchDeleteTickets.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useBatchDeleteMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2] }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchDeleteTickets.mockRejectedValue(new Error('del err'));
      const { result } = renderHook(() => useBatchDeleteMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1] }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchCloseMutation', () => {
    it('should close tickets on success', async () => {
      mockApi.batchCloseTickets.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useBatchCloseMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], closureReason: 'done' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchCloseTickets.mockRejectedValue(new Error('close err'));
      const { result } = renderHook(() => useBatchCloseMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1] }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchReopenMutation', () => {
    it('should reopen tickets on success', async () => {
      mockApi.batchReopenTickets.mockResolvedValue({ successCount: 1 } as any);
      const { result } = renderHook(() => useBatchReopenMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [5], reason: 'incomplete' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.batchReopenTickets.mockRejectedValue(new Error('reopen err'));
      const { result } = renderHook(() => useBatchReopenMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1] }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchExportMutation', () => {
    it('should export tickets and trigger download', async () => {
      const mockBlob = new Blob(['test']);
      mockApi.batchExportTickets.mockResolvedValue(mockBlob as any);
      const mockCreateObjectURL = jest.fn(() => 'blob:url');
      const mockRevokeObjectURL = jest.fn();
      Object.defineProperty(window, 'URL', { value: { createObjectURL: mockCreateObjectURL, revokeObjectURL: mockRevokeObjectURL }, writable: true });
      const mockAppendChild = jest.spyOn(document.body, 'appendChild').mockImplementation(n => n);
      const mockRemoveChild = jest.spyOn(document.body, 'removeChild').mockImplementation(n => n);

      const { result } = renderHook(() => useBatchExportMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ ticketIds: [1, 2], config: { format: 'csv', fileName: 'test.csv', fields: [] } }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockCreateObjectURL).toHaveBeenCalled();
      expect(message.success).toHaveBeenCalled();
      mockAppendChild.mockRestore();
      mockRemoveChild.mockRestore();
    });

    it('should handle export error', async () => {
      mockApi.batchExportTickets.mockRejectedValue(new Error('export fail'));
      const { result } = renderHook(() => useBatchExportMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ config: { format: 'csv', fileName: 'x.csv', fields: [] } }); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalled();
    });
  });

  describe('useUndoBatchOperationMutation', () => {
    it('should undo operation on success', async () => {
      mockApi.undoBatchOperation.mockResolvedValue({ successCount: 2 } as any);
      const { result } = renderHook(() => useUndoBatchOperationMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('op-1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle undo error', async () => {
      mockApi.undoBatchOperation.mockRejectedValue(new Error('undo err'));
      const { result } = renderHook(() => useUndoBatchOperationMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('op-x'); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchOperationMutation', () => {
    it('should execute batch operation on success', async () => {
      mockApi.executeBatchOperation.mockResolvedValue({ successCount: 5, failedCount: 1 } as any);
      const { result } = renderHook(() => useBatchOperationMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ type: 'assign', ticketIds: [1, 2, 3, 4, 5, 6] } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });

    it('should handle batch operation error', async () => {
      mockApi.executeBatchOperation.mockRejectedValue(new Error('batch err'));
      const { result } = renderHook(() => useBatchOperationMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ type: 'assign', ticketIds: [1] } as any); });
      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchOperationProgressQuery', () => {
    it('should fetch progress', async () => {
      mockApi.getBatchOperationProgress.mockResolvedValue({ operationId: 'op-1', progress: 50, status: 'running' } as any);
      const { result } = renderHook(() => useBatchOperationProgressQuery('op-1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getBatchOperationProgress).toHaveBeenCalledWith('op-1');
    });

    it('should not fetch when operationId is empty', () => {
      renderHook(() => useBatchOperationProgressQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getBatchOperationProgress).not.toHaveBeenCalled();
    });

    it('should respect enabled option', () => {
      renderHook(() => useBatchOperationProgressQuery('op-1', { enabled: false }), { wrapper: createWrapper() });
      expect(mockApi.getBatchOperationProgress).not.toHaveBeenCalled();
    });
  });

  describe('useBatchOperationLogsQuery', () => {
    it('should fetch logs', async () => {
      mockApi.getBatchOperationLogs.mockResolvedValue({ items: [], total: 0 } as any);
      const { result } = renderHook(() => useBatchOperationLogsQuery({ page: 1, pageSize: 10 }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getBatchOperationLogs).toHaveBeenCalledWith({ page: 1, pageSize: 10 });
    });
  });

  describe('useBatchOperationStatsQuery', () => {
    it('should fetch stats', async () => {
      mockApi.getBatchOperationStats.mockResolvedValue({ total: 100 } as any);
      const { result } = renderHook(() => useBatchOperationStatsQuery({ groupBy: 'day' }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getBatchOperationStats).toHaveBeenCalledWith({ groupBy: 'day' });
    });
  });

  describe('useBatchOperationPermissionsQuery', () => {
    it('should fetch permissions', async () => {
      mockApi.getBatchOperationPermissions.mockResolvedValue({ canAssign: true } as any);
      const { result } = renderHook(() => useBatchOperationPermissionsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useScheduledOperationsQuery', () => {
    it('should fetch scheduled operations', async () => {
      mockApi.getScheduledOperations.mockResolvedValue([{ id: 's1' }] as any);
      const { result } = renderHook(() => useScheduledOperationsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useMyBatchOperationStatsQuery', () => {
    it('should fetch my stats', async () => {
      mockApi.getMyBatchOperationStats.mockResolvedValue({ total: 5 } as any);
      const { result } = renderHook(() => useMyBatchOperationStatsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useExportStatusQuery', () => {
    it('should fetch export status', async () => {
      mockApi.getExportStatus.mockResolvedValue({ status: 'completed' } as any);
      const { result } = renderHook(() => useExportStatusQuery('exp-1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getExportStatus).toHaveBeenCalledWith('exp-1');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useExportStatusQuery('exp-1', false), { wrapper: createWrapper() });
      expect(mockApi.getExportStatus).not.toHaveBeenCalled();
    });

    it('should not fetch when exportId is empty', () => {
      renderHook(() => useExportStatusQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getExportStatus).not.toHaveBeenCalled();
    });
  });
});
