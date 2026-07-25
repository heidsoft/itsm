import { BPMNWorkflowApi } from '@/lib/api/bpmn-workflow-api';
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
const mockDelete = httpClient.delete as jest.Mock;

describe('BPMNWorkflowApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('migrateLegacyApprovalWorkflow', () => {
    it('should post migration request', async () => {
      const result = { workflowId: 1, processDefinitionKey: 'key1', skipped: false };
      mockPost.mockResolvedValue({ data: result });
      const res = await BPMNWorkflowApi.migrateLegacyApprovalWorkflow(1, true);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/approval-workflows/1/migrate-to-bpmn?dryRun=true', {});
      expect(res).toEqual(result);
    });
  });

  describe('createProcessDefinition', () => {
    it('should post to process-definitions', async () => {
      const data = { key: 'k1', name: 'n1', xml: '<xml/>' };
      const expected = { id: 1, key: 'k1', name: 'n1', version: 1, status: 'draft', xml: '<xml/>' };
      mockPost.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.createProcessDefinition(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', data);
      expect(res).toEqual(expected);
    });
  });

  describe('listProcessDefinitions', () => {
    it('should get with query params', async () => {
      const response = { data: { items: [], total: 0, page: 1, pageSize: 10 } };
      mockGet.mockResolvedValue(response);
      const res = await BPMNWorkflowApi.listProcessDefinitions({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', { page: '1', pageSize: '10' });
      expect(res).toEqual(response.data);
    });
  });

  describe('getProcessDefinition', () => {
    it('should get by key', async () => {
      const expected = { id: 1, key: 'k1', name: 'n1', version: 1, status: 'active', xml: '' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getProcessDefinition('k1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1');
      expect(res).toEqual(expected);
    });
  });

  describe('updateProcessDefinition', () => {
    it('should put update data', async () => {
      const data = { name: 'updated' };
      const expected = { id: 1, key: 'k1', name: 'updated', version: 1, status: 'active', xml: '' };
      mockPut.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.updateProcessDefinition('k1', data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1', data);
      expect(res).toEqual(expected);
    });
  });

  describe('deleteProcessDefinition', () => {
    it('should delete by key', async () => {
      mockDelete.mockResolvedValue(undefined);
      await BPMNWorkflowApi.deleteProcessDefinition('k1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1');
    });
  });

  describe('exportProcessDefinition', () => {
    it('should get export XML', async () => {
      mockGet.mockResolvedValue('<xml/>');
      const res = await BPMNWorkflowApi.exportProcessDefinition('k1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1/export');
      expect(res).toBe('<xml/>');
    });
  });

  describe('cloneProcessDefinition', () => {
    it('should post clone request', async () => {
      const data = { newKey: 'k2', newName: 'Clone' };
      const expected = { id: 2, key: 'k2', name: 'Clone', version: 1, status: 'draft', xml: '' };
      mockPost.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.cloneProcessDefinition('k1', data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1/clone', data);
      expect(res).toEqual(expected);
    });
  });

  describe('setProcessDefinitionActive', () => {
    it('should put active state', async () => {
      const expected = { id: 1, key: 'k1', name: 'n1', version: 1, status: 'active', xml: '' };
      mockPut.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.setProcessDefinitionActive('k1', true);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1/active', { active: true });
      expect(res).toEqual(expected);
    });
  });

  describe('startProcess', () => {
    it('should post start request', async () => {
      const data = { processDefinitionKey: 'k1' };
      const expected = { id: 'inst1', processDefinitionKey: 'k1', status: 'running' };
      mockPost.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.startProcess(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', data);
      expect(res).toEqual(expected);
    });
  });

  describe('listProcessInstances', () => {
    it('should get with params', async () => {
      const response = { data: { items: [], total: 0, page: 1, pageSize: 10 } };
      mockGet.mockResolvedValue(response);
      const res = await BPMNWorkflowApi.listProcessInstances({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', { page: '1', pageSize: '10' });
      expect(res).toEqual(response.data);
    });
  });

  describe('getProcessInstance', () => {
    it('should get by id', async () => {
      const expected = { id: 'inst1', status: 'running' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getProcessInstance('inst1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1');
      expect(res).toEqual(expected);
    });
  });

  describe('setProcessInstanceVariables', () => {
    it('should put variables', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.setProcessInstanceVariables('inst1', { foo: 'bar' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1/variables', { variables: { foo: 'bar' } });
    });
  });

  describe('suspendProcess', () => {
    it('should put suspend', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.suspendProcess('inst1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1/suspend', {});
    });
  });

  describe('resumeProcess', () => {
    it('should put resume', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.resumeProcess('inst1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1/resume', {});
    });
  });

  describe('terminateProcess', () => {
    it('should put terminate', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.terminateProcess('inst1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1/terminate', {});
    });
  });

  describe('listUserTasks', () => {
    it('should get tasks with params', async () => {
      const response = { data: { items: [], total: 0, page: 1, pageSize: 10 } };
      mockGet.mockResolvedValue(response);
      const res = await BPMNWorkflowApi.listUserTasks({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/tasks', { page: '1', pageSize: '10' });
      expect(res).toEqual(response.data);
    });
  });

  describe('getTask', () => {
    it('should get task by id', async () => {
      const expected = { id: 't1', name: 'Task' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getTask('t1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1');
      expect(res).toEqual(expected);
    });
  });

  describe('claimTask', () => {
    it('should post claim', async () => {
      mockPost.mockResolvedValue(undefined);
      await BPMNWorkflowApi.claimTask('t1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/claim', {});
    });
  });

  describe('assignTask', () => {
    it('should put assign', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.assignTask('t1', { assignee: 5 });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/assign', { assignee: 5 });
    });
  });

  describe('completeTask', () => {
    it('should put complete', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.completeTask('t1', { comment: 'done' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/complete', { comment: 'done' });
    });
  });

  describe('submitApprovalDecision', () => {
    it('should post decision', async () => {
      const data = { action: 'approve' as const, comment: 'ok' };
      mockPost.mockResolvedValue(undefined);
      await BPMNWorkflowApi.submitApprovalDecision('t1', data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/decisions', data);
    });
  });

  describe('getApprovalHistory', () => {
    it('should get approval history', async () => {
      const expected = [{ id: 1, action: 'approve' }];
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getApprovalHistory('inst1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/inst1/approval-history');
      expect(res).toEqual(expected);
    });
  });

  describe('cancelTask', () => {
    it('should put cancel', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.cancelTask('t1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/cancel', {});
    });
  });

  describe('setTaskVariables', () => {
    it('should put task variables', async () => {
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.setTaskVariables('t1', { x: 1 });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/variables', { variables: { x: 1 } });
    });
  });

  describe('createCounterSignTask', () => {
    it('should post counter-sign', async () => {
      const data = { users: [1, 2], type: 'parallel' as const, voteType: 'agree' as const };
      const expected = [{ id: 'cs1' }];
      mockPost.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.createCounterSignTask('t1', data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/counter-sign', data);
      expect(res).toEqual(expected);
    });
  });

  describe('getCounterSignStatus', () => {
    it('should get counter-sign status', async () => {
      const expected = { mainTaskId: 't1', total: 3, approved: 2, rejected: 0, abstained: 0, completed: false };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getCounterSignStatus('t1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/counter-sign-status');
      expect(res).toEqual(expected);
    });
  });

  describe('vote', () => {
    it('should put vote', async () => {
      const data = { vote: 'agree' as const, comment: 'ok' };
      mockPut.mockResolvedValue(undefined);
      await BPMNWorkflowApi.vote('t1', data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/tasks/t1/vote', data);
    });
  });

  describe('getInstanceStats', () => {
    it('should get instance stats', async () => {
      const expected = { totalInstances: 10, runningInstances: 5 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getInstanceStats({ processDefinitionKey: 'k1' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/stats/instances', { processDefinitionKey: 'k1' });
      expect(res).toEqual(expected);
    });
  });

  describe('getTaskStats', () => {
    it('should get task stats', async () => {
      const expected = { totalTasks: 20, pendingTasks: 5 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getTaskStats({});
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/stats/tasks', {});
      expect(res).toEqual(expected);
    });
  });

  describe('listVersions', () => {
    it('should get versions with key', async () => {
      const expected = { items: [], total: 0, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.listVersions('k1', { page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions', { page: '1', pageSize: '10', key: 'k1' });
      expect(res).toEqual(expected);
    });
  });

  describe('getVersion', () => {
    it('should get version by key and number', async () => {
      const expected = { key: 'k1', version: 2, name: 'v2' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getVersion('k1', 2);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions/k1/2');
      expect(res).toEqual(expected);
    });
  });

  describe('createVersion', () => {
    it('should post new version', async () => {
      const data = { key: 'k1', xml: '<xml/>' };
      const expected = { key: 'k1', version: 3 };
      mockPost.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.createVersion(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/versions', data);
      expect(res).toEqual(expected);
    });
  });

  describe('activateVersion', () => {
    it('should put activate', async () => {
      const expected = { key: 'k1', version: 2, isActivated: true };
      mockPut.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.activateVersion('k1', 2);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/versions/k1/2/activate', {});
      expect(res).toEqual(expected);
    });
  });

  describe('rollbackVersion', () => {
    it('should put rollback', async () => {
      const expected = { key: 'k1', version: 1, isActivated: true };
      mockPut.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.rollbackVersion('k1', 1);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/versions/k1/1/rollback', {});
      expect(res).toEqual(expected);
    });
  });

  describe('compareVersions', () => {
    it('should get version comparison', async () => {
      const expected = { key: 'k1', version1: 1, version2: 2, diff: { added: [], removed: [], modified: [] } };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.compareVersions('k1', 1, 2);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/versions/k1/compare', { version1: '1', version2: '2' });
      expect(res).toEqual(expected);
    });
  });

  describe('getVersionChangeLogs', () => {
    it('should get changelogs', async () => {
      const expected = { items: [], total: 0, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getVersionChangeLogs('k1', { page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/k1/changelogs', { key: 'k1', page: '1', pageSize: '10' });
      expect(res).toEqual(expected);
    });
  });

  describe('getVersionChangeLogById', () => {
    it('should get changelog by id', async () => {
      const expected = { id: 1, processDefinitionKey: 'k1', version: 1, changeType: 'update' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNWorkflowApi.getVersionChangeLogById(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/changelogs/1');
      expect(res).toEqual(expected);
    });
  });
});
