import { workflowEngineService } from '../workflow-engine-service';
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

describe('WorkflowEngineService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getWorkflowDefinitions', () => {
    it('should fetch definitions without params', async () => {
      const response = { data: [{ id: 1, key: 'approval', name: 'Approval Flow' }], total: 1, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue(response);

      const result = await workflowEngineService.getWorkflowDefinitions();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions');
      expect(result).toEqual(response);
    });

    it('should fetch definitions with query params', async () => {
      const response = { data: [], total: 0, page: 2, pageSize: 10 };
      mockGet.mockResolvedValue(response);

      await workflowEngineService.getWorkflowDefinitions({ page: 2, pageSize: 10, key: 'approval', category: 'hr' });

      expect(mockGet).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions?')
      );
      const url = (mockGet as jest.Mock).mock.calls[0][0] as string;
      expect(url).toContain('page=2');
      expect(url).toContain('pageSize=10');
      expect(url).toContain('key=approval');
      expect(url).toContain('category=hr');
    });
  });

  describe('getWorkflowDefinition', () => {
    it('should fetch a single definition by key', async () => {
      const definition = { id: 1, key: 'approval', name: 'Approval', version: '1.0' };
      mockGet.mockResolvedValue(definition);

      const result = await workflowEngineService.getWorkflowDefinition('approval');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/approval');
      expect(result).toEqual(definition);
    });

    it('should fetch a specific version', async () => {
      mockGet.mockResolvedValue({ id: 1, key: 'approval', version: '2.0' });

      await workflowEngineService.getWorkflowDefinition('approval', '2.0');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/approval?version=2.0');
    });
  });

  describe('createWorkflowDefinition', () => {
    it('should create a workflow definition', async () => {
      const data = { key: 'onboarding', name: 'Onboarding', bpmnXml: '<xml/>' };
      const response = { id: 2, ...data, version: '1.0', category: '', isActive: true, isLatest: true, tenantId: 1, createdAt: '', updatedAt: '' };
      mockPost.mockResolvedValue(response);

      const result = await workflowEngineService.createWorkflowDefinition(data);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions', data);
      expect(result.key).toBe('onboarding');
    });
  });

  describe('updateWorkflowDefinition', () => {
    it('should update a workflow definition', async () => {
      const data = { name: 'Updated Name' };
      const response = { id: 1, key: 'approval', name: 'Updated Name', version: '1.0' };
      mockPut.mockResolvedValue(response);

      const result = await workflowEngineService.updateWorkflowDefinition('approval', '1.0', data as any);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/bpmn/process-definitions/approval?version=1.0', data);
      expect(result.name).toBe('Updated Name');
    });
  });

  describe('setWorkflowDefinitionActive', () => {
    it('should activate a workflow definition', async () => {
      mockPut.mockResolvedValue(undefined);

      await workflowEngineService.setWorkflowDefinitionActive('approval', '1.0', true);

      expect(mockPut).toHaveBeenCalledWith(
        '/api/v1/bpmn/process-definitions/approval/active?version=1.0',
        { active: true }
      );
    });

    it('should deactivate a workflow definition', async () => {
      mockPut.mockResolvedValue(undefined);

      await workflowEngineService.setWorkflowDefinitionActive('approval', '1.0', false);

      expect(mockPut).toHaveBeenCalledWith(
        '/api/v1/bpmn/process-definitions/approval/active?version=1.0',
        { active: false }
      );
    });
  });

  describe('getWorkflowInstances', () => {
    it('should fetch instances without params', async () => {
      const response = { data: [{ id: 1, processInstanceId: 'pi-1' }], total: 1, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue(response);

      const result = await workflowEngineService.getWorkflowInstances();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances');
      expect(result).toEqual(response);
    });

    it('should fetch instances with query params', async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });

      await workflowEngineService.getWorkflowInstances({ processDefinitionKey: 'approval', status: 'running', page: 1, pageSize: 20 });

      const url = (mockGet as jest.Mock).mock.calls[0][0] as string;
      expect(url).toContain('processDefinitionKey=approval');
      expect(url).toContain('status=running');
    });
  });

  describe('startWorkflowInstance', () => {
    it('should start a new workflow instance', async () => {
      const response = { id: 1, processInstanceId: 'pi-new', processDefinitionKey: 'approval', businessKey: 'TKT-001', status: 'running', startTime: '', variables: {} };
      mockPost.mockResolvedValue(response);

      const result = await workflowEngineService.startWorkflowInstance('approval', 'TKT-001', { priority: 'high' });

      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', {
        processDefinitionKey: 'approval',
        businessKey: 'TKT-001',
        variables: { priority: 'high' },
      });
      expect(result.processInstanceId).toBe('pi-new');
    });

    it('should start without variables', async () => {
      mockPost.mockResolvedValue({ id: 1, processInstanceId: 'pi-2' });

      await workflowEngineService.startWorkflowInstance('onboarding', 'EMP-001');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/bpmn/process-instances', {
        processDefinitionKey: 'onboarding',
        businessKey: 'EMP-001',
        variables: {},
      });
    });
  });

  describe('getWorkflowInstance', () => {
    it('should get a specific instance by id', async () => {
      const instance = { id: 1, processInstanceId: 'pi-1', status: 'running' };
      mockGet.mockResolvedValue(instance);

      const result = await workflowEngineService.getWorkflowInstance('pi-1');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/process-instances/pi-1');
      expect(result.processInstanceId).toBe('pi-1');
    });
  });

  describe('getUserTasks', () => {
    it('should fetch user tasks without params', async () => {
      const response = { data: [{ id: 1, taskId: 't-1', taskName: 'Review' }], total: 1 };
      mockGet.mockResolvedValue(response);

      const result = await workflowEngineService.getUserTasks();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/workflow/tasks');
      expect(result).toEqual(response);
    });

    it('should fetch user tasks with params', async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });

      await workflowEngineService.getUserTasks({ page: 1, assignee: 'alice', status: 'pending' });

      const url = (mockGet as jest.Mock).mock.calls[0][0] as string;
      expect(url).toContain('assignee=alice');
      expect(url).toContain('status=pending');
    });
  });

  describe('completeTask', () => {
    it('should complete a task with variables', async () => {
      mockPut.mockResolvedValue(undefined);

      await workflowEngineService.completeTask('t-1', { approved: true });

      expect(mockPut).toHaveBeenCalledWith('/api/v1/workflow/tasks/t-1/complete', { variables: { approved: true } });
    });

    it('should complete a task without variables', async () => {
      mockPut.mockResolvedValue(undefined);

      await workflowEngineService.completeTask('t-2');

      expect(mockPut).toHaveBeenCalledWith('/api/v1/workflow/tasks/t-2/complete', { variables: undefined });
    });
  });

  describe('assignTask', () => {
    it('should assign a task to a user', async () => {
      mockPut.mockResolvedValue(undefined);

      await workflowEngineService.assignTask('t-1', 'bob');

      expect(mockPut).toHaveBeenCalledWith('/api/v1/workflow/tasks/t-1/claim', { assignee: 'bob' });
    });
  });
});
