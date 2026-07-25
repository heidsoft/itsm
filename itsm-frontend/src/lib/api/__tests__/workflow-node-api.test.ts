import { WorkflowNodeApi } from '@/lib/api/workflow-node-api';
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
const mockPut = httpClient.put as jest.Mock;

describe('WorkflowNodeApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getNodeInstances', () => {
    it('should get tasks for instance and transform', async () => {
      const backendRes = [
        { id: 't1', taskId: 'task-1', name: 'Review', status: 'pending', assignee: '5', assigneeName: 'John', createdTime: '2024-01-01T00:00:00Z' },
      ];
      mockGet.mockResolvedValue(backendRes);
      const res = await WorkflowNodeApi.getNodeInstances('inst-1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/tasks?processInstanceId=inst-1');
      expect(res).toHaveLength(1);
      expect(res[0].nodeId).toBe('task-1');
      expect(res[0].nodeName).toBe('Review');
      expect(res[0].assignee).toBe(5);
    });

    it('should return empty array on error', async () => {
      mockGet.mockRejectedValue(new Error('Network error'));
      const res = await WorkflowNodeApi.getNodeInstances('inst-1');
      expect(res).toEqual([]);
    });

    it('should handle null response', async () => {
      mockGet.mockResolvedValue(null);
      const res = await WorkflowNodeApi.getNodeInstances('inst-1');
      expect(res).toEqual([]);
    });
  });

  describe('completeNode', () => {
    it('should put complete with variables', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowNodeApi.completeNode({ instanceId: 'inst-1', nodeId: 'task-1', output: { approved: true } });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task-1/complete', { variables: { approved: true } });
    });
  });

  describe('skipNode', () => {
    it('should complete with _skipped flag', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowNodeApi.skipNode('inst-1', 'task-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task-1/complete', { variables: { _skipped: true } });
    });
  });

  describe('retryNode', () => {
    it('should complete with _retried flag', async () => {
      mockPut.mockResolvedValue(undefined);
      await WorkflowNodeApi.retryNode('inst-1', 'task-1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task-1/complete', { variables: { _retried: true } });
    });
  });

  describe('listWorkflowTasks', () => {
    it('should return empty array', async () => {
      const res = await WorkflowNodeApi.listWorkflowTasks('inst-1');
      expect(res).toEqual([]);
    });
  });
});
