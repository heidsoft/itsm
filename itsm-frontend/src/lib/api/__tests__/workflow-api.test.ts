import { WorkflowApi } from '@/lib/api/workflow-api';
import { httpClient } from '@/lib/api/http-client';

// Mock httpClient methods directly
jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    getTenantId: jest.fn().mockReturnValue(null),
    setToken: jest.fn(),
    getAuthToken: jest.fn().mockReturnValue(null),
  },
}));

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('WorkflowApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getWorkflows', () => {
    it('should fetch workflows successfully', async () => {
      const mockData = [
        {
          id: 1,
          key:'approvalWorkflow',
          name: 'Approval Workflow',
          description: 'Test workflow',
          version: 1,
          createdAt: '2024-01-01T10:00:00Z',
          updatedAt: '2024-01-01T10:00:00Z',
        },
      ];

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getWorkflows();

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions'),
        expect.any(Object)
      );

      expect(result.workflows).toHaveLength(1);
      expect(result.workflows[0].name).toBe('Approval Workflow');
    });

    it('should handle empty workflow list', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([]);

      const result = await WorkflowApi.getWorkflows();

      expect(result.workflows).toHaveLength(0);
      expect(result.total).toBe(0);
    });

    it('should pass pagination parameters correctly', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([]);

      await WorkflowApi.getWorkflows({ page: 2, pageSize: 10 });

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ page: 2, pageSize: 10 })
      );
    });
  });

  describe('getWorkflow', () => {
    it('should fetch single workflow successfully', async () => {
      const mockData = {
        id: 1,
        key:'approvalWorkflow',
        name: 'Approval Workflow',
        description: 'Test workflow',
        version: 1,
        createdAt: '2024-01-01T10:00:00Z',
        updatedAt: '2024-01-01T10:00:00Z',
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getWorkflow('approval_workflow');

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow')
      );

      expect(result.name).toBe('Approval Workflow');
    });

    it('should handle workflow not found', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('Workflow not found'));

      await expect(WorkflowApi.getWorkflow('nonexistent')).rejects.toBeDefined();
    });
  });

  describe('getProcessDefinition (backward compatible)', () => {
    it('should fetch process definition using key', async () => {
      const mockData = {
        id: 1,
        key:'testProcess',
        name: 'Test Process',
        version: 1,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getProcessDefinition('test_process');

      expect(result.name).toBe('Test Process');
    });
  });

  describe('getProcessVersions', () => {
    it('should fetch all versions of a process', async () => {
      const mockData = [
        { id: 1, key:'testProcess', version: 1, name: 'Test Process v1' },
        { id: 2, key:'testProcess', version: 2, name: 'Test Process v2' },
      ];

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.getProcessVersions('test_process');

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/versions')
      );

      expect(result).toHaveLength(2);
    });
  });

	describe('workflow persistence', () => {
	  it('returns the already-unwrapped definition after update', async () => {
		const updated = { id: '12', code: 'incident_flow', name: 'Incident Flow', version: 2 };
		(httpClient.put as jest.Mock).mockResolvedValueOnce(updated);

		await expect(
		  WorkflowApi.updateWorkflow('incident_flow', { name: 'Incident Flow' }, '2')
		).resolves.toBe(updated);
		expect(httpClient.put).toHaveBeenCalledWith(
		  '/api/v1/bpmn/process-definitions/incident_flow?version=2',
		  expect.objectContaining({ name: 'Incident Flow' })
		);
	  });

	  it('persists a new version through the backend version endpoint', async () => {
		(httpClient.post as jest.Mock).mockResolvedValueOnce({ version: 2 });
		const payload = {
		  processDefinitionKey: 'incident_flow',
		  name: 'Incident Flow',
		  bpmnXml: '<definitions />',
		  changeLog: '创建新版本',
		};

		await WorkflowApi.createWorkflowVersion(payload);
		expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/versions', payload);
	  });
	});

  describe('listWorkflowInstances', () => {
    it('should fetch workflow instances', async () => {
      const mockData = [
        {
          id: '1',
          instanceId: '1',
          processDefinitionKey: 'approval_workflow',
          businessKey: 'ticket-123',
          status: 'running',
          startTime: '2024-01-01T10:00:00Z',
        },
      ];

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.listWorkflowInstances({});

      expect(httpClient.get).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.any(Object)
      );

      expect(result.instances).toHaveLength(1);
    });
  });

  describe('startWorkflow', () => {
    it('should start a new workflow', async () => {
      const mockData = {
        id: '1',
        processInstanceId: '1',
        processDefinitionKey: 'approval_workflow',
        businessKey: 'BIZ-123',
        status: 'running',
        startTime: '2024-01-01T10:00:00Z',
      };

      (httpClient.post as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await WorkflowApi.startWorkflow({
        workflowId: 'approval_workflow',
        variables: { ticketId: 123 },
      });

      expect(httpClient.post).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.any(Object)
      );

      expect(result.status).toBe('running');
    });
  });

  describe('suspendWorkflow', () => {
    it('should suspend a running workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.suspendWorkflow('1');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/suspend'),
        expect.any(Object)
      );
    });
  });

  describe('resumeWorkflow', () => {
    it('should resume a suspended workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.resumeWorkflow('1');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/resume')
      );
    });
  });

  describe('terminateWorkflow', () => {
    it('should terminate a running workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.terminateWorkflow('1', 'Terminated by user');

      expect(httpClient.put).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/terminate'),
        expect.any(Object)
      );
    });
  });

  describe('deleteWorkflow', () => {
    it('should delete a workflow', async () => {
      (httpClient.delete as jest.Mock).mockResolvedValueOnce(undefined);

      await WorkflowApi.deleteWorkflow('approval_workflow');

      expect(httpClient.delete).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow')
      );
    });
  });

  describe('createWorkflow', () => {
    it('should create a workflow', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'new_wf', name: 'New' });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(5);
      const result = await WorkflowApi.createWorkflow({ code: 'new_wf', name: 'New', type: 'ticket' as any });
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ key: 'new_wf', name: 'New', tenantId: 5 }));
    });

    it('should use default tenantId when none set', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(null);
      await WorkflowApi.createWorkflow({ name: 'Test' } as any);
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ tenantId: 1 }));
    });
  });

  describe('cloneWorkflow', () => {
    it('should clone by getting original and creating new', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'orig', name: 'Original', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' });
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 2, key: 'orig_copy' });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.cloneWorkflow('orig', 'Clone');
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'Clone' }));
    });
  });

  describe('activateWorkflow', () => {
    it('should activate with default version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateWorkflow('wf1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=1.0.0', { active: true });
    });

    it('should activate with specified version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateWorkflow('wf1', '2.0.0');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=2.0.0', { active: true });
    });
  });

  describe('deactivateWorkflow', () => {
    it('should deactivate workflow', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.deactivateWorkflow('wf1', '1.0.0');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1/active?version=1.0.0', { active: false });
    });
  });

  describe('validateWorkflow', () => {
    it('should return valid result', async () => {
      const result = await WorkflowApi.validateWorkflow({ name: 'Test' } as any);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });
  });

  describe('exportWorkflow', () => {
    it('should export workflow', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 1, key: 'wf1', name: 'WF', version: 1, bpmnXml: '<xml/>', createdAt: '2024-01-01', updatedAt: '2024-01-01' });
      const result = await WorkflowApi.exportWorkflow('wf1');
      expect(result.version).toBe('1.0');
      expect(result.workflow).toBeDefined();
    });
  });

  describe('importWorkflow', () => {
    it('should import workflow', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.importWorkflow({ version: '1.0', exportedAt: new Date(), exportedBy: 'sys', workflow: { code: 'imp', name: 'Imported', type: 'ticket' } as any });
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'Imported' }));
    });

    it('should throw on invalid data', async () => {
      await expect(WorkflowApi.importWorkflow({ version: '1.0', exportedAt: new Date(), exportedBy: 'sys' } as any)).rejects.toThrow('无效的导入数据');
    });
  });

  describe('publishVersion', () => {
    it('should return first version', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([{ id: 1, key: 'wf1', name: 'WF', version: 2, status: 'active', createdAt: '2024-01-01', updatedAt: '2024-01-01' }]);
      const result = await WorkflowApi.publishVersion('wf1');
      expect(result.version).toBe(2);
    });

    it('should throw if no versions', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([]);
      await expect(WorkflowApi.publishVersion('wf1')).rejects.toThrow('未找到可发布的版本');
    });
  });

  describe('activateVersion', () => {
    it('should activate a specific version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.activateVersion('wf1', 2);
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/2/activate', {});
    });
  });

  describe('rollbackVersion', () => {
    it('should rollback to version', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.rollbackVersion('wf1', 1, 'bug fix');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/1/rollback', { reason: 'bug fix' });
    });

    it('should use default reason', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.rollbackVersion('wf1', 1);
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/versions/wf1/1/rollback', { reason: '回滚操作' });
    });
  });

  describe('deleteVersion', () => {
    it('should delete a version', async () => {
      (httpClient.delete as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.deleteVersion('wf1', 2);
      expect(httpClient.delete).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/wf1?version=2');
    });
  });

  describe('compareVersions', () => {
    it('should compare and map fields', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ addedNodes: ['n1'], removedNodes: ['n2'], modifiedNodes: ['n3'] });
      const result = await WorkflowApi.compareVersions('wf1', 1, 2);
      expect(result.elementsAdded).toEqual(['n1']);
      expect(result.elementsRemoved).toEqual(['n2']);
      expect(result.elementsModified).toEqual(['n3']);
      expect(result.isIdentical).toBe(false);
    });

    it('should return identical when no changes', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({});
      const result = await WorkflowApi.compareVersions('wf1', 1, 2);
      expect(result.isIdentical).toBe(true);
    });

    it('should handle null response', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      const result = await WorkflowApi.compareVersions('wf1', 1, 2);
      expect(result.isIdentical).toBe(true);
    });
  });

  describe('getInstance', () => {
    it('should get single instance', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ id: 'inst1', instanceId: 'inst1', processDefinitionKey: 'wf1', status: 'running', startTime: '2024-01-01' });
      const result = await WorkflowApi.getInstance('inst1');
      expect(result.id).toBe('inst1');
      expect(result.status).toBe('running');
    });
  });

  describe('retryInstance', () => {
    it('should retry by resuming', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.retryInstance('inst1');
      expect(httpClient.put).toHaveBeenCalledWith(expect.stringContaining('/resume'));
    });
  });

  describe('completeNode', () => {
    it('should complete a node', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.completeNode({ instanceId: 'inst1', nodeId: 'task1', output: { approved: true } });
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { approved: true } });
    });
  });

  describe('skipNode', () => {
    it('should skip a node by completing with _skipped flag', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.skipNode('inst1', 'task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { _skipped: true } });
    });
  });

  describe('retryNode', () => {
    it('should retry a node', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.retryNode('inst1', 'task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/complete', { variables: { _retried: true } });
    });
  });

  describe('getTemplates', () => {
    it('should return empty array (mock)', async () => {
      const result = await WorkflowApi.getTemplates();
      expect(result).toEqual([]);
    });
  });

  describe('getTemplate', () => {
    it('should return mock template', async () => {
      const result = await WorkflowApi.getTemplate('t1');
      expect(result.id).toBe('t1');
      expect(result.name).toBe('未命名模板');
    });
  });

  describe('createFromTemplate', () => {
    it('should create from template', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ id: 1 });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await WorkflowApi.createFromTemplate('t1', 'New WF');
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', expect.objectContaining({ name: 'New WF' }));
    });
  });

  describe('saveAsTemplate', () => {
    it('should return mock template data', async () => {
      const result = await WorkflowApi.saveAsTemplate('wf1', { name: 'My Template', category: 'approval', tags: ['tag1'] });
      expect(result.name).toBe('My Template');
      expect(result.category).toBe('approval');
      expect(result.tags).toEqual(['tag1']);
    });
  });

  describe('getWorkflowStats', () => {
    it('should return mock stats', async () => {
      const result = await WorkflowApi.getWorkflowStats('wf1');
      expect(result.workflowId).toBe('wf1');
      expect(result.totalInstances).toBe(0);
    });
  });

  describe('getInstanceStats', () => {
    it('should get instance stats from backend', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ total: 10, running: 3, completed: 5, suspended: 1, terminated: 1 });
      const result = await WorkflowApi.getInstanceStats({ processDefinitionKey: 'wf1' });
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/stats/instances', expect.objectContaining({ processDefinitionKey: 'wf1' }));
      expect(result.total).toBe(10);
    });

    it('should return defaults when null', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      const result = await WorkflowApi.getInstanceStats();
      expect(result.total).toBe(0);
    });
  });

  describe('getTaskStats', () => {
    it('should get task stats from backend', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ totalTasks: 20, completedTasks: 15, pendingTasks: 5, overdueTasks: 2, averageCompletion: 3.5 });
      const result = await WorkflowApi.getTaskStats({ assignee: 'user1' });
      expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/stats/tasks', expect.objectContaining({ assignee: 'user1' }));
      expect(result.totalTasks).toBe(20);
    });

    it('should return defaults when null', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce(null);
      const result = await WorkflowApi.getTaskStats();
      expect(result.totalTasks).toBe(0);
    });
  });

  describe('createCounterSignTasks', () => {
    it('should create counter-sign tasks', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({ data: [{ taskId: 't1', assignee: 'user1', status: 'pending' }] });
      const result = await WorkflowApi.createCounterSignTasks('task1', ['user1', 'user2'], 'parallel', 2);
      expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/counter-sign', { approvalType: 'parallel', approvers: ['user1', 'user2'], threshold: 2 });
      expect(result).toHaveLength(1);
    });

    it('should handle empty response', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce({});
      const result = await WorkflowApi.createCounterSignTasks('task1', ['user1']);
      expect(result).toEqual([]);
    });
  });

  describe('getCounterSignStatus', () => {
    it('should get counter-sign status', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ data: { parentTaskId: 't1', total: 3, completed: 1, approved: 1, rejected: 0, pending: 2, status: 'pending' } });
      const result = await WorkflowApi.getCounterSignStatus('t1');
      expect(result.total).toBe(3);
    });

    it('should return defaults when no data', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({});
      const result = await WorkflowApi.getCounterSignStatus('t1');
      expect(result.total).toBe(0);
      expect(result.status).toBe('pending');
    });
  });

  describe('vote', () => {
    it('should vote on a task', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.vote('task1', true, 'Approved');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/vote', { approved: true, comment: 'Approved' });
    });
  });

  describe('getNodeStats', () => {
    it('should return empty array (mock)', async () => {
      const result = await WorkflowApi.getNodeStats('wf1');
      expect(result).toEqual([]);
    });
  });

  describe('getBottleneckAnalysis', () => {
    it('should return empty bottlenecks (mock)', async () => {
      const result = await WorkflowApi.getBottleneckAnalysis('wf1');
      expect(result.bottlenecks).toEqual([]);
    });
  });

  describe('listMyTasks', () => {
    it('should list my tasks with array response', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce([{ id: '1', taskName: 'Review', status: 'pending', assignee: '10' }]);
      const result = await WorkflowApi.listMyTasks({ page: 1, pageSize: 20 });
      expect(result.items).toHaveLength(1);
      expect(result.items[0].nodeName).toBe('Review');
    });

    it('should handle paginated response', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: '1', taskName: 'T1' }], total: 50, page: 2, size: 10 });
      const result = await WorkflowApi.listMyTasks({ page: 2, pageSize: 10 });
      expect(result.total).toBe(50);
      expect(result.page).toBe(2);
    });
  });

  describe('claimMyTask', () => {
    it('should claim a task', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce(undefined);
      await WorkflowApi.claimMyTask('task1');
      expect(httpClient.put).toHaveBeenCalledWith('/api/v1/bpmn/tasks/task1/claim');
    });
  });

  describe('listWorkflowTasks', () => {
    it('should list tasks for instance', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: '1', taskName: 'Step 1', status: 'completed' }] });
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      const result = await WorkflowApi.listWorkflowTasks('inst1');
      expect(result).toHaveLength(1);
    });

    it('should throw error on failure', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('fail'));
      (httpClient.getTenantId as jest.Mock).mockReturnValue(1);
      await expect(WorkflowApi.listWorkflowTasks('inst1')).rejects.toThrow('fail');
    });
  });

  describe('deployWorkflow', () => {
    it('should deploy by activating', async () => {
      (httpClient.put as jest.Mock).mockResolvedValueOnce({});
      await WorkflowApi.deployWorkflow('wf1');
      expect(httpClient.put).toHaveBeenCalledWith(expect.stringContaining('/active'), expect.any(Object));
    });

    it('should propagate error on deploy failure', async () => {
      (httpClient.put as jest.Mock).mockRejectedValueOnce(new Error('Deploy failed'));
      await expect(WorkflowApi.deployWorkflow('wf1')).rejects.toThrow('Deploy failed');
    });
  });

  describe('getNodeInstances', () => {
    it('should get node instances from tasks', async () => {
      (httpClient.get as jest.Mock)
        .mockResolvedValueOnce({ id: 'inst1', instanceId: 'inst1', processDefinitionKey: 'wf1', status: 'running', startTime: '2024-01-01' })
        .mockResolvedValueOnce([{ id: 't1', taskId: 'task1', name: 'Review', status: 'pending', assignee: '5' }]);
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result).toHaveLength(1);
      expect(result[0].nodeName).toBe('Review');
    });

    it('should return empty on error', async () => {
      (httpClient.get as jest.Mock).mockRejectedValueOnce(new Error('fail'));
      const result = await WorkflowApi.getNodeInstances('inst1');
      expect(result).toEqual([]);
    });
  });

  describe('exportReport', () => {
    it('should generate CSV blob', async () => {
      (httpClient.get as jest.Mock)
        .mockResolvedValueOnce({ id: 1, key: 'wf1', name: 'Report WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' })
        .mockResolvedValueOnce({ total: 5, running: 2, completed: 2, suspended: 0, terminated: 1 })
        .mockResolvedValueOnce({ totalTasks: 10, completedTasks: 8, pendingTasks: 2, overdueTasks: 0, averageCompletion: 2.5 });
      const result = await WorkflowApi.exportReport({ workflowId: 'wf1', format: 'excel', startDate: '2024-01-01', endDate: '2024-12-31' });
      expect(result).toBeInstanceOf(Blob);
    });
  });

  describe('getWorkflows with paginated response', () => {
    it('should handle {data: [...]} response format', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ data: [{ id: 1, key: 'wf', name: 'WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' }], total: 1 });
      const result = await WorkflowApi.getWorkflows();
      expect(result.workflows).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('should handle {items: [...]} response format', async () => {
      (httpClient.get as jest.Mock).mockResolvedValueOnce({ items: [{ id: 1, key: 'wf', name: 'WF', version: 1, createdAt: '2024-01-01', updatedAt: '2024-01-01' }], pagination: { total: 5 } });
      const result = await WorkflowApi.getWorkflows();
      expect(result.workflows).toHaveLength(1);
      expect(result.total).toBe(5);
    });
  });
});
