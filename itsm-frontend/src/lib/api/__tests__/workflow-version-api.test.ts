import { WorkflowVersionApi } from '../workflow-version-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

jest.mock('@/types/workflow', () => ({
  WorkflowType: { TICKET: 'ticket' },
  WorkflowStatus: { DRAFT: 'draft', ACTIVE: 'active' },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('WorkflowVersionApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getWorkflowVersions', () => {
    it('should get versions and transform response', async () => {
      const rawVersions = [
        { id: 1, key: 'proc1', name: 'Process 1', version: 2, status: 'active', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-02T00:00:00Z' },
      ];
      mockGet.mockResolvedValue(rawVersions);
      const result = await WorkflowVersionApi.getWorkflowVersions('proc1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions?process_key=proc1');
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('1');
      expect(result[0].code).toBe('proc1');
      expect(result[0].name).toBe('Process 1');
      expect(result[0].version).toBe(2);
      expect(result[0].nodes).toEqual([]);
      expect(result[0].connections).toEqual([]);
    });

    it('should handle non-array response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await WorkflowVersionApi.getWorkflowVersions('proc1');
      expect(result).toEqual([]);
    });

    it('should normalize missing fields', async () => {
      mockGet.mockResolvedValue([{}]);
      const result = await WorkflowVersionApi.getWorkflowVersions('proc1');
      expect(result[0].id).toBe('');
      expect(result[0].code).toBe('');
      expect(result[0].name).toBe('');
      expect(result[0].version).toBe(1);
    });
  });

  describe('publishVersion', () => {
    it('should return first version when versions exist', async () => {
      const rawVersions = [{ id: 1, key: 'proc1', name: 'V1', version: 1, status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' }];
      mockGet.mockResolvedValue(rawVersions);
      const result = await WorkflowVersionApi.publishVersion('proc1');
      expect(result.id).toBe('1');
    });

    it('should throw error when no versions found', async () => {
      mockGet.mockResolvedValue([]);
      await expect(WorkflowVersionApi.publishVersion('proc1')).rejects.toThrow('未找到可发布的版本');
    });
  });

  describe('activateVersion', () => {
    it('should activate version', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowVersionApi.activateVersion('proc1', 2);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/versions/proc1/2/activate', {});
    });
  });

  describe('rollbackVersion', () => {
    it('should rollback version with reason', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowVersionApi.rollbackVersion('proc1', 1, 'Bug in v2');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/versions/proc1/1/rollback', { reason: 'Bug in v2' });
    });

    it('should rollback version with default reason', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowVersionApi.rollbackVersion('proc1', 1);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/versions/proc1/1/rollback', { reason: '回滚操作' });
    });
  });

  describe('deleteVersion', () => {
    it('should delete version', async () => {
      mockDelete.mockResolvedValue(undefined);
      await WorkflowVersionApi.deleteVersion('proc1', 2);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/proc1?version=2');
    });
  });

  describe('compareVersions', () => {
    it('should compare versions and transform response', async () => {
      mockGet.mockResolvedValue({ addedNodes: ['n1'], removedNodes: ['n2'], modifiedNodes: ['n3'] });
      const result = await WorkflowVersionApi.compareVersions('proc1', 1, 2);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions/proc1/compare', { baseVersion: 1, targetVersion: 2 });
      expect(result.elementsAdded).toEqual(['n1']);
      expect(result.elementsRemoved).toEqual(['n2']);
      expect(result.elementsModified).toEqual(['n3']);
      expect(result.isIdentical).toBe(false);
    });

    it('should return identical when no differences', async () => {
      mockGet.mockResolvedValue({ addedNodes: [], removedNodes: [], modifiedNodes: [] });
      const result = await WorkflowVersionApi.compareVersions('proc1', 1, 2);
      expect(result.isIdentical).toBe(true);
    });

    it('should handle null response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await WorkflowVersionApi.compareVersions('proc1', 1, 2);
      expect(result.isIdentical).toBe(true);
      expect(result.elementsAdded).toEqual([]);
    });
  });

  describe('getProcessVersions (alias)', () => {
    it('should delegate to getWorkflowVersions', async () => {
      mockGet.mockResolvedValue([]);
      const result = await WorkflowVersionApi.getProcessVersions('proc1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions?process_key=proc1');
      expect(result).toEqual([]);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Version not found'));
      await expect(WorkflowVersionApi.getWorkflowVersions('x')).rejects.toThrow('Version not found');
    });
  });
});
