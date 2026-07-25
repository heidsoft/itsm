import { WorkflowDefinitionApi } from '@/lib/api/workflow-definition-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    getTenantId: jest.fn().mockReturnValue(1),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('WorkflowDefinitionApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getWorkflows', () => {
    it('should get workflow list and transform', async () => {
      const backendRes = [
        { id: 1, key: 'proc1', name: 'Process 1', version: 1, status: 'active', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' },
      ];
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowDefinitionApi.getWorkflows({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', { page: 1, pageSize: 10 });
      expect(res.workflows).toHaveLength(1);
      expect(res.workflows[0].code).toBe('proc1');
      expect(res.total).toBe(1);
    });

    it('should handle empty response', async () => {
      mockGet.mockResolvedValue(null);
      const res = await WorkflowDefinitionApi.getWorkflows();
      expect(res.workflows).toHaveLength(0);
      expect(res.total).toBe(0);
    });
  });

  describe('getWorkflow', () => {
    it('should get single workflow and transform', async () => {
      const backendRes = { id: 1, key: 'proc1', name: 'Process 1', version: 2, status: 'draft', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z' };
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowDefinitionApi.getWorkflow('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/1');
      expect(res.code).toBe('proc1');
      expect(res.version).toBe(2);
    });
  });

  describe('createWorkflow', () => {
    it('should post create request', async () => {
      const expected = { id: '1', code: 'proc1', name: 'New' };
      mockPost.mockResolvedValue(expected);
      const res = await WorkflowDefinitionApi.createWorkflow({ code: 'proc1', name: 'New', type: 'ticket' as any });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'New' }));
      expect(res).toEqual(expected);
    });
  });

  describe('updateWorkflow', () => {
    it('should put update request', async () => {
      const response = { code: 0, message: 'success', data: { id: '1', name: 'Updated' } };
      mockPut.mockResolvedValue(response);
      const res = await WorkflowDefinitionApi.updateWorkflow('1', { name: 'Updated' }, '2.0.0');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/1?version=2.0.0', expect.objectContaining({ name: 'Updated' }));
      expect(res).toEqual(response.data);
    });

    it('should throw on non-zero code', async () => {
      mockPut.mockResolvedValue({ code: 1, message: 'Error' });
      await expect(WorkflowDefinitionApi.updateWorkflow('1', { name: 'X' })).rejects.toThrow('Error');
    });
  });

  describe('deleteWorkflow', () => {
    it('should delete workflow', async () => {
      mockDelete.mockResolvedValue(undefined);
      await WorkflowDefinitionApi.deleteWorkflow('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/1?version=1.0.0');
    });
  });

  describe('activateWorkflow', () => {
    it('should activate workflow', async () => {
      mockPut.mockResolvedValue({ code: 0, message: 'ok' });
      await WorkflowDefinitionApi.activateWorkflow('1', '1.0.0');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/1/active?version=1.0.0', { active: true });
    });

    it('should throw on non-zero code', async () => {
      mockPut.mockResolvedValue({ code: 1, message: 'Failed' });
      await expect(WorkflowDefinitionApi.activateWorkflow('1')).rejects.toThrow('Failed');
    });
  });

  describe('deactivateWorkflow', () => {
    it('should deactivate workflow', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowDefinitionApi.deactivateWorkflow('1', '2.0.0');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/1/active?version=2.0.0', { active: false });
    });
  });

  describe('validateWorkflow', () => {
    it('should return valid result', async () => {
      const res = await WorkflowDefinitionApi.validateWorkflow({});
      expect(res).toEqual({ isValid: true, errors: [], warnings: [] });
    });
  });

  describe('deployWorkflow', () => {
    it('should call activateWorkflow', async () => {
      jest.spyOn(console, 'error').mockImplementation(() => {});
      mockPut.mockResolvedValue({ code: 0, message: 'ok' });
      await WorkflowDefinitionApi.deployWorkflow('1');
      expect(mockPut).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/process-definitions/1/active'), { active: true });
    });
  });
});
