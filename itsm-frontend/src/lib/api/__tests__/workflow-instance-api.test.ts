import { WorkflowInstanceApi } from '@/lib/api/workflow-instance-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;

describe('WorkflowInstanceApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('startWorkflow', () => {
    it('should post start request and transform response', async () => {
      const backendRes = { id: '123', processInstanceId: 'inst-1', processDefinitionKey: 'proc1', businessKey: 'BIZ-1', status: 'running', startTime: '2024-01-01T00:00:00Z' };
      mockPost.mockResolvedValue(backendRes);
      const res = await WorkflowInstanceApi.startWorkflow({ workflowId: 'proc1', variables: { x: 1 } });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', expect.objectContaining({ processDefinitionKey: 'proc1' }));
      expect(res.id).toBe('inst-1');
      expect(res.workflowId).toBe('proc1');
    });
  });

  describe('getInstances', () => {
    it('should get instances and transform', async () => {
      const backendRes = { data: [{ id: '1', instanceId: 'inst-1', processDefinitionKey: 'proc1', businessKey: 'B1', status: 'running', startTime: '2024-01-01T00:00:00Z' }], total: 1 };
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowInstanceApi.getInstances({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', expect.objectContaining({ page: 1, pageSize: 10 }));
      expect(res.instances).toHaveLength(1);
      expect(res.instances[0].id).toBe('inst-1');
    });

    it('should handle array response', async () => {
      const backendRes = [{ id: '1', instanceId: 'inst-1', processDefinitionKey: 'proc1', businessKey: 'B1', status: 'running', startTime: '2024-01-01T00:00:00Z' }];
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowInstanceApi.getInstances();
      expect(res.instances).toHaveLength(1);
    });
  });

  describe('getInstance', () => {
    it('should get single instance', async () => {
      const backendRes = { id: '1', instanceId: 'inst-1', processDefinitionKey: 'proc1', businessKey: 'B1', status: 'completed', startTime: '2024-01-01T00:00:00Z', endTime: '2024-01-02T00:00:00Z' };
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowInstanceApi.getInstance('inst-1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1');
      expect(res.id).toBe('inst-1');
      expect(res.status).toBe('completed');
    });
  });

  describe('cancelInstance', () => {
    it('should put terminate', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.cancelInstance('inst-1', 'test reason');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/terminate', { reason: 'test reason' });
    });
  });

  describe('suspendInstance', () => {
    it('should put suspend', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.suspendInstance('inst-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/suspend', { reason: 'User requested' });
    });
  });

  describe('resumeInstance', () => {
    it('should put resume', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.resumeInstance('inst-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/resume');
    });
  });

  describe('retryInstance', () => {
    it('should call resumeInstance', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.retryInstance('inst-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/resume');
    });
  });

  describe('listWorkflowInstances (alias)', () => {
    it('should delegate to getInstances', async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      const res = await WorkflowInstanceApi.listWorkflowInstances();
      expect(mockGet).toHaveBeenCalled();
      expect(res.instances).toBeDefined();
    });
  });

  describe('suspendWorkflow (alias)', () => {
    it('should delegate to suspendInstance', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.suspendWorkflow('inst-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/suspend', expect.any(Object));
    });
  });

  describe('terminateWorkflow (alias)', () => {
    it('should delegate to cancelInstance', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowInstanceApi.terminateWorkflow('inst-1', 'reason');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst-1/terminate', { reason: 'reason' });
    });
  });
});
