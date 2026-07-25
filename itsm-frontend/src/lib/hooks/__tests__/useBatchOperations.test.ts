import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useBatchAssignMutation,
  useBatchUpdateStatusMutation,
  useBatchDeleteMutation,
  useBatchOperationProgressQuery,
  BATCH_OPERATION_KEYS,
} from '../useBatchOperations';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock BatchOperationsApi
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
const mockBatchOperationsApi = BatchOperationsApi as jest.Mocked<typeof BatchOperationsApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useBatchOperations hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('BATCH_OPERATION_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(BATCH_OPERATION_KEYS.all).toEqual(['batch-operations']);
      expect(BATCH_OPERATION_KEYS.progress('op-1')).toEqual(['batch-operations', 'progress', 'op-1']);
    });
  });

  describe('useBatchAssignMutation', () => {
    it('should batch assign tickets', async () => {
      mockBatchOperationsApi.batchAssignTickets.mockResolvedValue({ successCount: 3 });

      const { result } = renderHook(() => useBatchAssignMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketIds: [1, 2, 3], assigneeId: 10 });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockBatchOperationsApi.batchAssignTickets).toHaveBeenCalledWith({
        ticketIds: [1, 2, 3],
        assigneeId: 10,
      });
    });
  });

  describe('useBatchUpdateStatusMutation', () => {
    it('should batch update status', async () => {
      mockBatchOperationsApi.batchUpdateStatus.mockResolvedValue({ successCount: 2 });

      const { result } = renderHook(() => useBatchUpdateStatusMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketIds: [1, 2], status: 'closed' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockBatchOperationsApi.batchUpdateStatus).toHaveBeenCalled();
    });
  });

  describe('useBatchDeleteMutation', () => {
    it('should batch delete tickets', async () => {
      mockBatchOperationsApi.batchDeleteTickets.mockResolvedValue({ successCount: 2 });

      const { result } = renderHook(() => useBatchDeleteMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketIds: [1, 2] });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockBatchOperationsApi.batchDeleteTickets).toHaveBeenCalledWith({ ticketIds: [1, 2] });
    });
  });

  describe('useBatchOperationProgressQuery', () => {
    it('should fetch progress', async () => {
      mockBatchOperationsApi.getBatchOperationProgress.mockResolvedValue({
        operationId: 'op-1',
        progress: 50,
        status: 'running',
      });

      const { result } = renderHook(
        () => useBatchOperationProgressQuery('op-1'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockBatchOperationsApi.getBatchOperationProgress).toHaveBeenCalledWith('op-1');
    });
  });
});
