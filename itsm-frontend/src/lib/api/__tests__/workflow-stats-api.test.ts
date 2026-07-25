import { WorkflowStatsApi } from '@/lib/api/workflow-stats-api';
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

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    exportReport: jest.fn().mockResolvedValue(new Blob(['test'])),
  },
}));

const mockGet = httpClient.get as jest.Mock;

describe('WorkflowStatsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getWorkflowStats', () => {
    it('should return default stats structure', async () => {
      const res = await WorkflowStatsApi.getWorkflowStats('wf1');
      expect(res).toEqual({
        workflowId: 'wf1',
        totalInstances: 0,
        runningInstances: 0,
        completedInstances: 0,
        failedInstances: 0,
        avgDuration: 0,
        successRate: 0,
        bottlenecks: [],
      });
    });
  });

  describe('getInstanceStats', () => {
    it('should get instance stats with params', async () => {
      const expected = { total: 100, running: 20, completed: 70, suspended: 5, terminated: 5 };
      mockGet.mockResolvedValue(expected);
      const res = await WorkflowStatsApi.getInstanceStats({ processDefinitionKey: 'proc1' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/stats/instances', { processDefinitionKey: 'proc1' });
      expect(res).toEqual(expected);
    });

    it('should unwrap data field', async () => {
      const expected = { total: 50, running: 10, completed: 30, suspended: 5, terminated: 5 };
      mockGet.mockResolvedValue({ code: 0, message: 'ok', data: expected });
      const res = await WorkflowStatsApi.getInstanceStats();
      expect(res).toEqual(expected);
    });
  });

  describe('getTaskStats', () => {
    it('should get task stats with params', async () => {
      const expected = { totalTasks: 200, completedTasks: 150, pendingTasks: 30, overdueTasks: 10, averageCompletion: 85 };
      mockGet.mockResolvedValue(expected);
      const res = await WorkflowStatsApi.getTaskStats({ processDefinitionKey: 'proc1' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/stats/tasks', { processDefinitionKey: 'proc1' });
      expect(res).toEqual(expected);
    });

    it('should unwrap data field', async () => {
      const expected = { totalTasks: 100, completedTasks: 80, pendingTasks: 15, overdueTasks: 5, averageCompletion: 90 };
      mockGet.mockResolvedValue({ code: 0, message: 'ok', data: expected });
      const res = await WorkflowStatsApi.getTaskStats();
      expect(res).toEqual(expected);
    });
  });

  describe('getNodeStats', () => {
    it('should return empty array', async () => {
      const res = await WorkflowStatsApi.getNodeStats('wf1');
      expect(res).toEqual([]);
    });
  });

  describe('getBottleneckAnalysis', () => {
    it('should return empty bottlenecks', async () => {
      const res = await WorkflowStatsApi.getBottleneckAnalysis('wf1');
      expect(res).toEqual({ bottlenecks: [] });
    });
  });

  describe('exportReport', () => {
    it('should delegate to WorkflowApi.exportReport', async () => {
      const res = await WorkflowStatsApi.exportReport({ workflowId: 'wf1', format: 'excel', startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(res).toBeInstanceOf(Blob);
    });
  });
});
