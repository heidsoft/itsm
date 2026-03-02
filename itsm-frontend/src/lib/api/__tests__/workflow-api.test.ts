import { WorkflowApi } from '@/lib/api/workflow-api';

// Mock fetch globally
global.fetch = jest.fn();

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('WorkflowApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockClear();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getWorkflows', () => {
    it('should fetch workflows successfully', async () => {
      // httpClient.get returns responseData.data directly
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [
          {
            id: 1,
            key: 'approval_workflow',
            name: 'Approval Workflow',
            description: 'Test workflow',
            version: 1,
            created_at: '2024-01-01T10:00:00Z',
            updated_at: '2024-01-01T10:00:00Z',
          },
        ],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.getWorkflows();

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.workflows).toHaveLength(1);
      expect(result.workflows[0].name).toBe('Approval Workflow');
    });

    it('should handle empty workflow list', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.getWorkflows();

      expect(result.workflows).toHaveLength(0);
      expect(result.total).toBe(0);
    });

    it('should pass pagination parameters correctly', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await WorkflowApi.getWorkflows({ page: 2, pageSize: 10 });

      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('page=2'), expect.any(Object));
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('page_size=10'),
        expect.any(Object)
      );
    });
  });

  describe('getWorkflow', () => {
    it('should fetch single workflow successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          key: 'approval_workflow',
          name: 'Approval Workflow',
          description: 'Test workflow',
          version: 1,
          created_at: '2024-01-01T10:00:00Z',
          updated_at: '2024-01-01T10:00:00Z',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.getWorkflow('approval_workflow');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.name).toBe('Approval Workflow');
    });

    it('should handle workflow not found', async () => {
      const mockResponse = {
        code: 4004,
        message: 'Workflow not found',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await expect(WorkflowApi.getWorkflow('nonexistent')).rejects.toBeDefined();
    });
  });

  describe('getProcessDefinition (backward compatible)', () => {
    it('should fetch process definition using key', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          key: 'test_process',
          name: 'Test Process',
          version: 1,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.getProcessDefinition('test_process');

      expect(result.name).toBe('Test Process');
    });
  });

  describe('getProcessVersions', () => {
    it('should fetch all versions of a process', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [
          {
            id: 1,
            key: 'test_process',
            version: 1,
            name: 'Test Process v1',
          },
          {
            id: 2,
            key: 'test_process',
            version: 2,
            name: 'Test Process v2',
          },
        ],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.getProcessVersions('test_process');

      // getProcessVersions calls getWorkflowVersions which uses /api/v1/bpmn/versions
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/versions'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result).toHaveLength(2);
    });
  });

  describe('listWorkflowInstances', () => {
    it('should fetch workflow instances', async () => {
      // httpClient.get returns responseData.data directly
      // The API expects instance_id and process_definition_key in snake_case
      const mockResponse = {
        code: 0,
        message: 'success',
        data: [
          {
            id: '1',
            instance_id: '1',
            process_definition_key: 'approval_workflow',
            business_key: 'ticket-123',
            status: 'running',
            start_time: '2024-01-01T10:00:00Z',
          },
        ],
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.listWorkflowInstances({});

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.instances).toHaveLength(1);
    });
  });

  describe('startWorkflow', () => {
    it('should start a new workflow', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          process_key: 'approval_workflow',
          status: 'running',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const result = await WorkflowApi.startWorkflow({
        workflowId: 'approval_workflow',
        variables: { ticketId: 123 },
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.status).toBe('running');
    });
  });

  describe('suspendWorkflow', () => {
    it('should suspend a running workflow', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await WorkflowApi.suspendWorkflow('1');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/suspend'),
        expect.objectContaining({
          method: 'PUT',
        })
      );
    });
  });

  describe('resumeWorkflow', () => {
    it('should resume a suspended workflow', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await WorkflowApi.resumeWorkflow('1');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/resume'),
        expect.objectContaining({
          method: 'PUT',
        })
      );
    });
  });

  describe('terminateWorkflow', () => {
    it('should terminate a running workflow', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await WorkflowApi.terminateWorkflow('1', 'Terminated by user');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-instances/1/terminate'),
        expect.objectContaining({
          method: 'PUT',
        })
      );
    });
  });

  describe('deleteWorkflow', () => {
    it('should delete a workflow', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await WorkflowApi.deleteWorkflow('approval_workflow');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/bpmn/process-definitions/approval_workflow'),
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });
});
