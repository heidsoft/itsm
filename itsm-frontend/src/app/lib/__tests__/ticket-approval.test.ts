import { TicketApprovalApi, ApprovalWorkflow, ApprovalRecord } from '@/lib/api/ticket-approval-api';

// Mock fetch globally
global.fetch = jest.fn();

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('TicketApprovalApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockClear();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getWorkflows', () => {
    it('should fetch approval workflows successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          items: [
            {
              id: 1,
              name: '技术审批流程',
              description: '技术工单审批流程',
              ticket_type: 'incident',
              priority: 'high',
              nodes: [],
              is_active: true,
              created_at: '2024-01-01T10:00:00Z',
              updated_at: '2024-01-01T10:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 20,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApprovalApi.getWorkflows({ ticketType: 'incident' });

      // Check that params were converted to snake_case in URL
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('ticket_type=incident'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.items).toHaveLength(1);
      expect(result.items[0].name).toBe('技术审批流程');
      // httpClient converts snake_case response to camelCase
      expect(result.items[0].ticketType).toBe('incident');
      expect(result.items[0].isActive).toBe(true);
    });

    it('should handle empty workflow list', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          items: [],
          total: 0,
          page: 1,
          page_size: 20,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApprovalApi.getWorkflows();

      expect(result.items).toHaveLength(0);
      expect(result.total).toBe(0);
    });
  });

  describe('createWorkflow', () => {
    it('should create approval workflow successfully', async () => {
      const workflowData: Partial<ApprovalWorkflow> = {
        name: '新审批流程',
        description: '测试审批流程',
        ticketType: 'incident',
        nodes: [],
      };

      // Mock response from backend (snake_case)
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          name: '新审批流程',
          description: '测试审批流程',
          ticket_type: 'incident',
          nodes: [],
          is_active: true,
          created_at: '2024-01-01T10:00:00Z',
          updated_at: '2024-01-01T10:00:00Z',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const result = await TicketApprovalApi.createWorkflow(workflowData);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/workflows'),
        expect.objectContaining({
          method: 'POST',
          // Request body should be converted to snake_case by httpClient
          body: expect.stringContaining('"ticket_type":"incident"'),
        })
      );

      expect(result.id).toBe(1);
      expect(result.name).toBe('新审批流程');
      expect(result.ticketType).toBe('incident');
    });
  });

  describe('submitApproval', () => {
    it('should submit approval successfully', async () => {
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

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 1,
        action: 'approve',
        comment: '审批通过',
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/submit'),
        expect.objectContaining({
          method: 'POST',
          // Request body should be converted to snake_case
          body: expect.stringContaining('"ticket_id":1'),
        })
      );
    });

    it('should handle rejection action', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ code: 0, message: 'success', data: null }),
      });

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 2,
        action: 'reject',
        comment: '不符合要求',
      });

      const callArgs = (fetch as jest.Mock).mock.calls[0];
      const body = JSON.parse(callArgs[1].body);
      expect(body.action).toBe('reject');
      expect(body.ticket_id).toBe(1);
    });

    it('should handle delegation action', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ code: 0, message: 'success', data: null }),
      });

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 3,
        action: 'delegate',
        comment: '转交处理',
        delegateToUserId: 100,
      });

      const callArgs = (fetch as jest.Mock).mock.calls[0];
      const body = JSON.parse(callArgs[1].body);
      expect(body.action).toBe('delegate');
      expect(body.delegate_to_user_id).toBe(100);
    });
  });

  describe('getApprovalRecords', () => {
    it('getApprovalRecords returns correct data', async () => {
      const mockResponse = {
        code: 200,
        message: 'success',
        data: {
          items: [
            {
              id: 1,
              ticket_id: 100,
              workflow_id: 1,
              approver_name: '张三',
              status: 'pending',
            },
          ],
          total: 1,
          page: 1,
          page_size: 10,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApprovalApi.getApprovalRecords({ ticketId: 100 });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('ticket_id=100'),
        expect.objectContaining({
          method: 'GET',
        })
      );
      
      expect(result.items[0].approverName).toBe('张三');
    });

    it('should filter by status', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          items: [],
          total: 0,
          page: 1,
          page_size: 20,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await TicketApprovalApi.getApprovalRecords({ status: 'pending' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('status=pending'),
        expect.objectContaining({
          method: 'GET',
        })
      );
    });
  });
});