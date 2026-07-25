import { TicketApprovalApi } from '@/lib/api/ticket-approval-api';
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

describe('TicketApprovalApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getWorkflows', () => {
    it('should fetch workflows without params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      const result = await TicketApprovalApi.getWorkflows();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/approval-workflows', {});
      expect(result.items).toEqual([]);
    });

    it('should pass query params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      await TicketApprovalApi.getWorkflows({ ticketType: 'incident', isActive: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/approval-workflows', expect.objectContaining({ ticketType: 'incident', isActive: true }));
    });
  });

  describe('createWorkflow', () => {
    it('should create a workflow', async () => {
      const data = { name: 'New Workflow', nodes: [], isActive: true };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await TicketApprovalApi.createWorkflow(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/approval-workflows', data);
      expect(result.id).toBe(1);
    });
  });

  describe('updateWorkflow', () => {
    it('should update a workflow', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      const result = await TicketApprovalApi.updateWorkflow(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/approval-workflows/1', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteWorkflow', () => {
    it('should delete a workflow', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApprovalApi.deleteWorkflow(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/approval-workflows/1');
    });
  });

  describe('getApprovalRecords', () => {
    it('should fetch approval records', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      const result = await TicketApprovalApi.getApprovalRecords({ ticketId: 5 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/approval/records', expect.objectContaining({ ticketId: 5 }));
      expect(result.items).toEqual([]);
    });
  });

  describe('submitApproval', () => {
    it('should submit approval action', async () => {
      const data = { ticketId: 1, approvalId: 2, action: 'approve' as const, comment: 'LGTM' };
      mockPost.mockResolvedValue(undefined);
      await TicketApprovalApi.submitApproval(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/approve', data);
    });
  });
});
