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

      const result = await TicketApprovalApi.getWorkflows({ ticket_type: 'incident' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/workflows'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.items).toHaveLength(1);
      expect(result.items[0].name).toBe('技术审批流程');
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
        ticket_type: 'incident',
        nodes: [],
      };

      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          ...workflowData,
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
          body: JSON.stringify(workflowData),
        })
      );

      expect(result.id).toBe(1);
      expect(result.name).toBe('新审批流程');
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
        ticket_id: 1,
        approval_id: 1,
        action: 'approve',
        comment: '审批通过',
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/submit'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            ticket_id: 1,
            approval_id: 1,
            action: 'approve',
            comment: '审批通过',
          }),
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
        ticket_id: 1,
        approval_id: 2,
        action: 'reject',
        comment: '不符合要求',
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/submit'),
        expect.objectContaining({
          method: 'POST',
        })
      );
      // Verify the body contains the correct action
      const callArgs = (fetch as jest.Mock).mock.calls[0];
      const body = JSON.parse(callArgs[1].body);
      expect(body.action).toBe('reject');
    });

    it('should handle delegation action', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ code: 0, message: 'success', data: null }),
      });

      await TicketApprovalApi.submitApproval({
        ticket_id: 1,
        approval_id: 3,
        action: 'delegate',
        comment: '转交处理',
        delegate_to_user_id: 100,
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/submit'),
        expect.objectContaining({
          method: 'POST',
        })
      );
      // Verify the body contains the correct action and delegate_to_user_id
      const callArgs = (fetch as jest.Mock).mock.calls[0];
      const body = JSON.parse(callArgs[1].body);
      expect(body.action).toBe('delegate');
      expect(body.delegate_to_user_id).toBe(100);
    });
  });

  describe('getApprovalRecords', () => {
    it('should fetch approval records successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          items: [
            {
              id: 1,
              ticket_id: 1,
              ticket_number: 'INC-001',
              ticket_title: '系统故障',
              workflow_id: 1,
              workflow_name: '技术审批',
              current_level: 1,
              total_levels: 2,
              approver_id: 10,
              approver_name: '张三',
              status: 'approved',
              action: 'approve',
              comment: '已处理',
              created_at: '2024-01-01T10:00:00Z',
              processed_at: '2024-01-01T11:00:00Z',
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

      const result = await TicketApprovalApi.getApprovalRecords({ ticket_id: 1 });

      expect(result.items).toHaveLength(1);
      expect(result.items[0].status).toBe('approved');
      expect(result.items[0].approver_name).toBe('张三');
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

      // GET request params are included in the URL
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tickets/approval/records?status=pending'),
        expect.objectContaining({
          method: 'GET',
        })
      );
    });
  });
});
