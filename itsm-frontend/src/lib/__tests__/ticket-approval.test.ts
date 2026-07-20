import type { ApprovalWorkflow } from '@/lib/api/ticket-approval-api';
import { TicketApprovalApi } from '@/lib/api/ticket-approval-api';
import { httpClient } from '@/lib/api/http-client';

// Path contract: these are the routes the backend actually registers
// (see itsm-backend/router/router.go L564). If a frontend URL diverges from
// this list, the test must fail — never mirror the SUT bug.
const APPROVAL_WORKFLOWS_PATH = '/api/v1/approval-workflows';
const APPROVAL_RECORDS_PATH = '/api/v1/tickets/approval/records';
const SUBMIT_APPROVAL_PATH = '/api/v1/tickets/workflow/approve';

// Mock httpClient methods directly
jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
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

describe('TicketApprovalApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getWorkflows', () => {
    it('should fetch approval workflows successfully', async () => {
      const mockData = {
        items: [
          {
            id: 1,
            name: '技术审批流程',
            description: '技术工单审批流程',
            ticketType: 'incident',
            priority: 'high',
            nodes: [],
            isActive: true,
            createdAt: '2024-01-01T10:00:00Z',
            updatedAt: '2024-01-01T10:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 20,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await TicketApprovalApi.getWorkflows({ ticketType: 'incident' });

      // Assert exact path against contract. StringContains would let the SUT
      // define its own (possibly wrong) path — that is the bug we just fixed.
      expect(httpClient.get).toHaveBeenCalledWith(APPROVAL_WORKFLOWS_PATH, expect.any(Object));

      expect(result.items).toHaveLength(1);
      expect(result.items[0].name).toBe('技术审批流程');
    });

    it('should handle empty workflow list', async () => {
      const mockData = {
        items: [],
        total: 0,
        page: 1,
        pageSize: 20,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

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

      const mockData = {
        id: 1,
        name: '新审批流程',
        description: '测试审批流程',
        ticketType: 'incident',
        nodes: [],
        isActive: true,
        createdAt: '2024-01-01T10:00:00Z',
        updatedAt: '2024-01-01T10:00:00Z',
      };

      (httpClient.post as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await TicketApprovalApi.createWorkflow(workflowData);

      expect(httpClient.post).toHaveBeenCalledWith(APPROVAL_WORKFLOWS_PATH, expect.any(Object));

      expect(result.id).toBe(1);
      expect(result.name).toBe('新审批流程');
    });
  });

  describe('submitApproval', () => {
    it('should submit approval successfully', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce(null);

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 1,
        action: 'approve',
        comment: '审批通过',
      });

      expect(httpClient.post).toHaveBeenCalledWith(SUBMIT_APPROVAL_PATH, expect.any(Object));
    });

    it('should handle rejection action', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce(null);

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 2,
        action: 'reject',
        comment: '不符合要求',
      });

      const callArgs = (httpClient.post as jest.Mock).mock.calls[0];
      const body = callArgs[1];
      expect(body.action).toBe('reject');
    });

    it('should handle delegation action', async () => {
      (httpClient.post as jest.Mock).mockResolvedValueOnce(null);

      await TicketApprovalApi.submitApproval({
        ticketId: 1,
        approvalId: 3,
        action: 'delegate',
        comment: '转交处理',
        delegateToUserId: 100,
      });

      const callArgs = (httpClient.post as jest.Mock).mock.calls[0];
      const body = callArgs[1];
      expect(body.action).toBe('delegate');
    });
  });

  describe('getApprovalRecords', () => {
    it('getApprovalRecords returns correct data', async () => {
      const mockData = {
        items: [
          {
            id: 1,
            ticketId: 100,
            workflowId: 1,
            approverName: '张三',
            status: 'pending',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      const result = await TicketApprovalApi.getApprovalRecords({ ticketId: 100 });

      expect(httpClient.get).toHaveBeenCalledWith(APPROVAL_RECORDS_PATH, expect.any(Object));

      expect(result).toHaveProperty('items');
      expect(Array.isArray(result.items)).toBe(true);
      expect(result.items.length).toBeGreaterThan(0);
    });

    it('should filter by status', async () => {
      const mockData = {
        items: [],
        total: 0,
        page: 1,
        pageSize: 20,
      };

      (httpClient.get as jest.Mock).mockResolvedValueOnce(mockData);

      await TicketApprovalApi.getApprovalRecords({ status: 'pending' });

      // httpClient.get receives (endpoint, paramsObj) where params include status
      expect(httpClient.get).toHaveBeenCalledWith(
        APPROVAL_RECORDS_PATH,
        expect.objectContaining({ status: 'pending' })
      );
    });
  });
});
