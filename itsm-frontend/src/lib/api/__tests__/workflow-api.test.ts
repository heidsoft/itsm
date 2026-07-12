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
});
