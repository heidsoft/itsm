import { BatchOperationsApi } from '../batch-operations-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('BatchOperationsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('executeBatchOperation', () => {
    it('should execute batch operation', async () => {
      mockPost.mockResolvedValue({ operationId: 'op1', status: 'completed' });
      await BatchOperationsApi.executeBatchOperation({ operationType: 'assign', ticketIds: [1, 2] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/execute', expect.any(Object));
    });
  });

  describe('validateBatchOperation', () => {
    it('should validate', async () => {
      mockPost.mockResolvedValue({ isValid: true });
      await BatchOperationsApi.validateBatchOperation({ operationType: 'close', ticketIds: [1] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/validate', expect.any(Object));
    });
  });

  describe('previewBatchOperation', () => {
    it('should preview', async () => {
      mockPost.mockResolvedValue({ affectedTickets: 5 });
      await BatchOperationsApi.previewBatchOperation({ operationType: 'assign', ticketIds: [1] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/preview', expect.any(Object));
    });
  });

  describe('batchAssignTickets', () => {
    it('should batch assign', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchAssignTickets({ ticketIds: [1, 2], assigneeId: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/assign', { ticketIds: [1, 2], assigneeId: 5 });
    });
  });

  describe('batchAssignRoundRobin', () => {
    it('should round robin assign', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchAssignRoundRobin({ ticketIds: [1, 2], teamId: 1 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/assign/round-robin', { ticketIds: [1, 2], teamId: 1 });
    });
  });

  describe('batchAssignLoadBalance', () => {
    it('should load balance assign', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchAssignLoadBalance({ ticketIds: [1], teamId: 2 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/assign/load-balance', { ticketIds: [1], teamId: 2 });
    });
  });

  describe('batchUpdateStatus', () => {
    it('should batch update status', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUpdateStatus({ ticketIds: [1], status: 'closed' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/status', { ticketIds: [1], status: 'closed' });
    });
  });

  describe('batchCloseTickets', () => {
    it('should batch close', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchCloseTickets({ ticketIds: [1, 2], closureReason: 'resolved' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/close', expect.objectContaining({ ticketIds: [1, 2] }));
    });
  });

  describe('batchReopenTickets', () => {
    it('should batch reopen', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchReopenTickets({ ticketIds: [1], reason: 'not fixed' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/reopen', { ticketIds: [1], reason: 'not fixed' });
    });
  });

  describe('batchUpdatePriority', () => {
    it('should batch update priority', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUpdatePriority({ ticketIds: [1], priority: 'high' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/priority', { ticketIds: [1], priority: 'high' });
    });
  });

  describe('batchUpdateType', () => {
    it('should batch update type', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUpdateType({ ticketIds: [1], type: 'incident' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/type', { ticketIds: [1], type: 'incident' });
    });
  });

  describe('batchUpdateCategory', () => {
    it('should batch update category', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUpdateCategory({ ticketIds: [1], categoryId: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/category', { ticketIds: [1], categoryId: 5 });
    });
  });

  describe('batchUpdateFields', () => {
    it('should batch update fields', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUpdateFields({ ticketIds: [1], customFields: { env: 'prod' } });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/fields', { ticketIds: [1], customFields: { env: 'prod' } });
    });
  });

  describe('batchAddTags', () => {
    it('should batch add tags', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchAddTags({ ticketIds: [1], tags: ['bug'] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/tags/add', { ticketIds: [1], tags: ['bug'] });
    });
  });

  describe('batchRemoveTags', () => {
    it('should batch remove tags', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchRemoveTags({ ticketIds: [1], tags: ['bug'] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/tags/remove', { ticketIds: [1], tags: ['bug'] });
    });
  });

  describe('batchReplaceTags', () => {
    it('should batch replace tags', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchReplaceTags({ ticketIds: [1], tags: ['feature'] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/tags/replace', { ticketIds: [1], tags: ['feature'] });
    });
  });

  describe('batchDeleteTickets', () => {
    it('should batch delete', async () => {
      mockRequest.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchDeleteTickets({ ticketIds: [1, 2], reason: 'duplicate' });
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/tickets/batch/delete', data: { ticketIds: [1, 2], reason: 'duplicate' } });
    });
  });

  describe('batchArchiveTickets', () => {
    it('should batch archive', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchArchiveTickets({ ticketIds: [1] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/archive', { ticketIds: [1] });
    });
  });

  describe('batchUnarchiveTickets', () => {
    it('should batch unarchive', async () => {
      mockPost.mockResolvedValue({ status: 'completed' });
      await BatchOperationsApi.batchUnarchiveTickets({ ticketIds: [1] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/unarchive', { ticketIds: [1] });
    });
  });

  describe('batchExportTickets', () => {
    it('should export', async () => {
      const blob = new Blob(['csv']);
      mockRequest.mockResolvedValue(blob);
      const result = await BatchOperationsApi.batchExportTickets({ ticketIds: [1], config: {} as any });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/tickets/batch/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('getExportStatus', () => {
    it('should get export status', async () => {
      mockGet.mockResolvedValue({ status: 'completed', downloadUrl: '/download' });
      await BatchOperationsApi.getExportStatus('exp1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/export/exp1/status');
    });
  });

  describe('getBatchOperationProgress', () => {
    it('should get progress', async () => {
      mockGet.mockResolvedValue({ progress: 50 });
      await BatchOperationsApi.getBatchOperationProgress('op1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/progress');
    });
  });

  describe('pauseBatchOperation', () => {
    it('should pause', async () => {
      mockPost.mockResolvedValue(undefined);
      await BatchOperationsApi.pauseBatchOperation('op1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/pause');
    });
  });

  describe('resumeBatchOperation', () => {
    it('should resume', async () => {
      mockPost.mockResolvedValue(undefined);
      await BatchOperationsApi.resumeBatchOperation('op1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/resume');
    });
  });

  describe('cancelBatchOperation', () => {
    it('should cancel', async () => {
      mockPost.mockResolvedValue(undefined);
      await BatchOperationsApi.cancelBatchOperation('op1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/cancel');
    });
  });

  describe('getBatchOperationLogs', () => {
    it('should get logs', async () => {
      mockGet.mockResolvedValue({ logs: [], total: 0 });
      await BatchOperationsApi.getBatchOperationLogs({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/logs', { page: 1 });
    });
  });

  describe('getBatchOperationStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalOperations: 100 });
      await BatchOperationsApi.getBatchOperationStats({ groupBy: 'day' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/stats', { groupBy: 'day' });
    });
  });

  describe('getBatchOperationPermissions', () => {
    it('should get permissions', async () => {
      mockGet.mockResolvedValue({ canAssign: true });
      await BatchOperationsApi.getBatchOperationPermissions();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/permissions');
    });
  });

  describe('canExecuteBatchOperation', () => {
    it('should check can execute', async () => {
      mockPost.mockResolvedValue({ canExecute: true });
      await BatchOperationsApi.canExecuteBatchOperation({ operationType: 'assign', ticketIds: [1] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/can-execute', { operationType: 'assign', ticketIds: [1] });
    });
  });

  describe('undoBatchOperation', () => {
    it('should undo operation', async () => {
      mockPost.mockResolvedValue({ status: 'undone' });
      await BatchOperationsApi.undoBatchOperation('op1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/undo');
    });
  });

  describe('canUndoBatchOperation', () => {
    it('should check can undo', async () => {
      mockGet.mockResolvedValue({ canUndo: true });
      await BatchOperationsApi.canUndoBatchOperation('op1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/op1/can-undo');
    });
  });

  describe('scheduleBatchOperation', () => {
    it('should schedule', async () => {
      mockPost.mockResolvedValue({ scheduleId: 's1' });
      await BatchOperationsApi.scheduleBatchOperation({ request: {} as any, scheduledAt: '2024-01-01' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/schedule', expect.any(Object));
    });
  });

  describe('cancelScheduledOperation', () => {
    it('should cancel schedule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await BatchOperationsApi.cancelScheduledOperation('s1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/schedule/s1');
    });
  });

  describe('getScheduledOperations', () => {
    it('should get scheduled operations', async () => {
      mockGet.mockResolvedValue([]);
      await BatchOperationsApi.getScheduledOperations();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/batch/operations/schedule');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockPost.mockRejectedValue(new Error('Forbidden'));
      await expect(BatchOperationsApi.executeBatchOperation({} as any)).rejects.toThrow('Forbidden');
    });
  });
});
