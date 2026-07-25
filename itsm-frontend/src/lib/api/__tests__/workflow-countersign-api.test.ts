import { WorkflowCounterSignApi } from '../workflow-countersign-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;

describe('WorkflowCounterSignApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('createCounterSignTasks', () => {
    it('should create parallel counter-sign tasks', async () => {
      const tasks = [{ taskId: 't1', assignee: 'user1', status: 'pending' }];
      mockPost.mockResolvedValue({ code: 0, message: 'ok', data: tasks });
      const result = await WorkflowCounterSignApi.createCounterSignTasks('task1', ['user1', 'user2'], 'parallel');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign', {
        approvalType: 'parallel',
        approvers: ['user1', 'user2'],
        threshold: 2,
      });
      expect(result).toEqual(tasks);
    });

    it('should create serial counter-sign tasks with custom threshold', async () => {
      mockPost.mockResolvedValue({ code: 0, message: 'ok', data: [] });
      await WorkflowCounterSignApi.createCounterSignTasks('task1', ['u1', 'u2', 'u3'], 'serial', 2);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign', {
        approvalType: 'serial',
        approvers: ['u1', 'u2', 'u3'],
        threshold: 2,
      });
    });

    it('should default to parallel and use approvers length as threshold', async () => {
      mockPost.mockResolvedValue({ code: 0, message: 'ok', data: [] });
      await WorkflowCounterSignApi.createCounterSignTasks('task1', ['u1', 'u2']);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign', {
        approvalType: 'parallel',
        approvers: ['u1', 'u2'],
        threshold: 2,
      });
    });

    it('should return empty array when response data is null', async () => {
      mockPost.mockResolvedValue({ code: 0, message: 'ok', data: null });
      const result = await WorkflowCounterSignApi.createCounterSignTasks('task1', ['u1']);
      expect(result).toEqual([]);
    });
  });

  describe('getCounterSignStatus', () => {
    it('should get counter-sign status', async () => {
      const status = { parentTaskId: 'task1', total: 3, completed: 1, approved: 1, rejected: 0, pending: 2, status: 'pending' };
      mockGet.mockResolvedValue({ code: 0, message: 'ok', data: status });
      const result = await WorkflowCounterSignApi.getCounterSignStatus('task1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign-status');
      expect(result).toEqual(status);
    });

    it('should return defaults when response data is null', async () => {
      mockGet.mockResolvedValue({ code: 0, message: 'ok', data: null });
      const result = await WorkflowCounterSignApi.getCounterSignStatus('task1');
      expect(result).toEqual({
        parentTaskId: 'task1',
        total: 0,
        completed: 0,
        approved: 0,
        rejected: 0,
        pending: 0,
        status: 'pending',
      });
    });
  });

  describe('vote', () => {
    it('should vote approve with comment', async () => {
      mockPut.mockResolvedValue({ code: 0, message: 'ok' });
      await WorkflowCounterSignApi.vote('task1', true, 'LGTM');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/vote', { approved: true, comment: 'LGTM' });
    });

    it('should vote reject without comment', async () => {
      mockPut.mockResolvedValue({ code: 0, message: 'ok' });
      await WorkflowCounterSignApi.vote('task1', false);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/vote', { approved: false, comment: undefined });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockPost.mockRejectedValue(new Error('Task not found'));
      await expect(WorkflowCounterSignApi.createCounterSignTasks('x', ['u1'])).rejects.toThrow('Task not found');
    });
  });
});
