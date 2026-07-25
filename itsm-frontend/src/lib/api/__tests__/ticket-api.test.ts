import { TicketApi } from '../ticket-api';
import { httpClient } from '../http-client';
import { handleApiRequest } from '../base-api-handler';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

jest.mock('../base-api-handler', () => ({
  handleApiRequest: jest.fn((promise) => promise),
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockPatch = httpClient.patch as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('TicketApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTickets', () => {
    it('should fetch tickets with params', async () => {
      const resp = { items: [], total: 0, size: 10 };
      mockGet.mockResolvedValue(resp);
      const result = await TicketApi.getTickets({ page: 1, pageSize: 10 });
      expect(handleApiRequest).toHaveBeenCalled();
      expect(result.size).toBe(10);
    });

    it('should default size to 20 when not provided', async () => {
      const resp = { items: [] };
      mockGet.mockResolvedValue(resp);
      const result = await TicketApi.getTickets();
      expect(result.size).toBe(20);
    });
  });

  describe('createTicket', () => {
    it('should create ticket', async () => {
      const data = { title: 'Test', description: 'Desc' };
      const expected = { id: 1, title: 'Test' };
      mockPost.mockResolvedValue(expected);
      const result = await TicketApi.createTicket(data as any);
      expect(result).toEqual(expected);
    });
  });

  describe('getTicket', () => {
    it('should get ticket by id', async () => {
      const expected = { id: 1, title: 'Test' };
      mockGet.mockResolvedValue(expected);
      const result = await TicketApi.getTicket(1);
      expect(result).toEqual(expected);
    });
  });

  describe('updateTicketStatus', () => {
    it('should update status', async () => {
      const expected = { id: 1, status: 'closed' };
      mockPut.mockResolvedValue(expected);
      const result = await TicketApi.updateTicketStatus(1, 'closed');
      expect(result).toEqual(expected);
    });
  });

  describe('updateTicket', () => {
    it('should update ticket', async () => {
      const data = { title: 'Updated' };
      const expected = { id: 1, title: 'Updated' };
      mockPut.mockResolvedValue(expected);
      const result = await TicketApi.updateTicket(1, data as any);
      expect(result).toEqual(expected);
    });
  });

  describe('deleteTicket', () => {
    it('should delete ticket', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApi.deleteTicket(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1');
    });
  });

  describe('approveTicket', () => {
    it('should approve ticket', async () => {
      const data = { action: 'approve' as const, ticketId: 1 };
      mockPost.mockResolvedValue({ success: true, message: 'ok' });
      const result = await TicketApi.approveTicket(1, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/approve', data);
      expect(result.success).toBe(true);
    });
  });

  describe('addComment', () => {
    it('should add comment', async () => {
      const expected = { id: 1, ticketId: 1, content: 'hello' };
      mockPost.mockResolvedValue(expected);
      const result = await TicketApi.addComment(1, 'hello');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/comments', { content: 'hello' });
      expect(result.content).toBe('hello');
    });
  });

  describe('assignTicket', () => {
    it('should assign with number', async () => {
      mockPost.mockResolvedValue({ id: 1, assigneeId: 5 });
      await TicketApi.assignTicket(1, 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/assign', { assigneeId: 5 });
    });

    it('should assign with object', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.assignTicket(1, { assigneeId: 5, comment: 'note' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/assign', { assigneeId: 5, comment: 'note' });
    });
  });

  describe('escalateTicket', () => {
    it('should escalate with string reason', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.escalateTicket(1, 'urgent');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/escalate', { reason: 'urgent' });
    });

    it('should escalate with object', async () => {
      const data = { level: 'L2', reason: 'complex' };
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.escalateTicket(1, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/escalate', data);
    });
  });

  describe('resolveTicket', () => {
    it('should resolve with string', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'resolved' });
      await TicketApi.resolveTicket(1, 'fixed');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/resolve', { resolution: 'fixed' });
    });

    it('should resolve with object', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.resolveTicket(1, { solution: 'patched', resolutionCode: 'RC1' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/resolve', { resolution: 'patched', resolutionCode: 'RC1' });
    });
  });

  describe('closeTicket', () => {
    it('should close with string', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.closeTicket(1, 'satisfied');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/close', { feedback: 'satisfied' });
    });

    it('should close with object', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.closeTicket(1, { closeNotes: 'done' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/close', { feedback: 'done' });
    });

    it('should close without feedback', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.closeTicket(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/close', {});
    });
  });

  describe('searchTickets', () => {
    it('should search tickets', async () => {
      mockGet.mockResolvedValue([{ id: 1 }]);
      const result = await TicketApi.searchTickets('test');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/search', { q: 'test' });
      expect(result).toHaveLength(1);
    });
  });

  describe('getOverdueTickets', () => {
    it('should get overdue tickets', async () => {
      mockGet.mockResolvedValue([]);
      await TicketApi.getOverdueTickets();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/overdue');
    });
  });

  describe('getSubtasks', () => {
    it('should return tickets array', async () => {
      mockGet.mockResolvedValue({ tickets: [{ id: 2 }] });
      const result = await TicketApi.getSubtasks(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/subtasks');
      expect(result).toEqual([{ id: 2 }]);
    });
  });

  describe('createSubtask', () => {
    it('should create subtask', async () => {
      mockPost.mockResolvedValue({ id: 2 });
      await TicketApi.createSubtask(1, { title: 'sub' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/subtasks', { title: 'sub', parentTicketId: 1 });
    });
  });

  describe('updateSubtask', () => {
    it('should update subtask', async () => {
      mockPatch.mockResolvedValue({ id: 2 });
      await TicketApi.updateSubtask(1, 2, { title: 'updated' } as any);
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/tickets/1/subtasks/2', { title: 'updated' });
    });
  });

  describe('deleteSubtask', () => {
    it('should delete subtask', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApi.deleteSubtask(1, 2);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/subtasks/2');
    });
  });

  describe('getTicketsByAssignee', () => {
    it('should get tickets by assignee', async () => {
      mockGet.mockResolvedValue([]);
      await TicketApi.getTicketsByAssignee(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assignee/5');
    });
  });

  describe('getTicketActivity', () => {
    it('should get activity', async () => {
      mockGet.mockResolvedValue([]);
      await TicketApi.getTicketActivity(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/activity');
    });
  });

  describe('getTicketComments', () => {
    it('should get comments', async () => {
      mockGet.mockResolvedValue({ comments: [], total: 0 });
      await TicketApi.getTicketComments(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/comments');
    });
  });

  describe('addTicketComment', () => {
    it('should add comment with data', async () => {
      const data = { content: 'hi', isInternal: true };
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.addTicketComment(1, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/comments', data);
    });
  });

  describe('updateTicketComment', () => {
    it('should update comment', async () => {
      const data = { content: 'updated' };
      mockPut.mockResolvedValue({ id: 10 });
      await TicketApi.updateTicketComment(1, 10, data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/1/comments/10', data);
    });
  });

  describe('deleteTicketComment', () => {
    it('should delete comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApi.deleteTicketComment(1, 10);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/comments/10');
    });
  });

  describe('getTicketAttachments', () => {
    it('should get attachments', async () => {
      mockGet.mockResolvedValue({ attachments: [], total: 0 });
      await TicketApi.getTicketAttachments(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/attachments');
    });
  });

  describe('uploadTicketAttachment', () => {
    it('should upload attachment', async () => {
      const file = new File(['data'], 'test.txt');
      mockPost.mockResolvedValue({ id: 1, fileName: 'test.txt' });
      await TicketApi.uploadTicketAttachment(1, file);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/attachments', expect.any(FormData), expect.any(Object));
    });
  });

  describe('getAttachmentDownloadUrl', () => {
    it('should return url', () => {
      expect(TicketApi.getAttachmentDownloadUrl(1, 5)).toBe('/api/v1/tickets/1/attachments/5');
    });
  });

  describe('getAttachmentPreviewUrl', () => {
    it('should return preview url', () => {
      expect(TicketApi.getAttachmentPreviewUrl(1, 5)).toBe('/api/v1/tickets/1/attachments/5/preview');
    });
  });

  describe('deleteTicketAttachment', () => {
    it('should delete attachment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApi.deleteTicketAttachment(1, 5);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/attachments/5');
    });
  });

  describe('getTicketWorkflow', () => {
    it('should get workflow state', async () => {
      mockGet.mockResolvedValue({ ticketId: 1, currentStatus: 'open' });
      await TicketApi.getTicketWorkflow(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/workflow/state');
    });
  });

  describe('acceptTicket', () => {
    it('should accept ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.acceptTicket(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/accept', { ticketId: 1 });
    });
  });

  describe('rejectTicket', () => {
    it('should reject ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.rejectTicket(1, 'bad');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/reject', { ticketId: 1, reason: 'bad' });
    });
  });

  describe('withdrawTicket', () => {
    it('should withdraw ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.withdrawTicket(1, 'mistake');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/withdraw', { ticketId: 1, reason: 'mistake' });
    });
  });

  describe('forwardTicket', () => {
    it('should forward ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.forwardTicket(1, 5, 'please help');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/forward', { ticketId: 1, toUserId: 5, comment: 'please help' });
    });
  });

  describe('ccTicket', () => {
    it('should cc ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.ccTicket(1, [2, 3], 'fyi', ['email']);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/cc', { ticketId: 1, ccUsers: [2, 3], comment: 'fyi', notifyChannels: ['email'] });
    });
  });

  describe('getMyCCRecords', () => {
    it('should get my cc records', async () => {
      mockGet.mockResolvedValue({ records: [], total: 0 });
      await TicketApi.getMyCCRecords();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/cc/my');
    });
  });

  describe('getTicketCCRecords', () => {
    it('should get ticket cc records', async () => {
      mockGet.mockResolvedValue({ records: [], total: 0 });
      await TicketApi.getTicketCCRecords(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/cc');
    });
  });

  describe('reopenTicket', () => {
    it('should reopen ticket', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await TicketApi.reopenTicket(1, 'not fixed');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/workflow/reopen', { ticketId: 1, reason: 'not fixed' });
    });
  });

  describe('updateWorkflowStep', () => {
    it('should update workflow step', async () => {
      mockPut.mockResolvedValue({ id: 10 });
      await TicketApi.updateWorkflowStep(1, 10, { status: 'completed' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/1/workflow/10', { status: 'completed' });
    });
  });

  describe('addTicketTags', () => {
    it('should add tags', async () => {
      mockPost.mockResolvedValue({ success: true, ticketId: 1, tags: ['bug'] });
      await TicketApi.addTicketTags(1, ['bug']);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/tags', { tags: ['bug'] });
    });
  });

  describe('removeTicketTags', () => {
    it('should remove tags via request', async () => {
      mockRequest.mockResolvedValue({ success: true });
      await TicketApi.removeTicketTags(1, ['bug']);
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/tickets/1/tags', data: { tags: ['bug'] } });
    });
  });

  describe('getTicketHistory', () => {
    it('should get history', async () => {
      mockGet.mockResolvedValue([]);
      await TicketApi.getTicketHistory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/history');
    });
  });

  describe('batchDeleteTickets', () => {
    it('should batch delete', async () => {
      mockRequest.mockResolvedValue(undefined);
      await TicketApi.batchDeleteTickets([1, 2]);
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/tickets/batch-delete', data: { ticketIds: [1, 2] } });
    });
  });

  describe('getTicketStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ total: 10, open: 5 });
      const result = await TicketApi.getTicketStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/stats');
      expect(result.total).toBe(10);
    });
  });

  describe('exportTickets', () => {
    it('should export tickets', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      const result = await TicketApi.exportTickets({ format: 'csv' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', url: '/api/v1/tickets/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('batchUpdateTickets', () => {
    it('should batch update', async () => {
      mockPost.mockResolvedValue(undefined);
      await TicketApi.batchUpdateTickets([1, 2], 'assign', { assigneeId: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/batch-assign', { ticketIds: [1, 2], action: 'assign', data: { assigneeId: 5 } });
    });
  });

  describe('getTemplates', () => {
    it('should get templates', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await TicketApi.getTemplates({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates', { page: 1 });
    });
  });

  describe('getTemplate', () => {
    it('should get template by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'T1' });
      await TicketApi.getTemplate(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates/1');
    });
  });

  describe('createTemplate', () => {
    it('should create template', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TicketApi.createTemplate({ name: 'T1' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/templates', { name: 'T1' });
    });
  });

  describe('updateTemplate', () => {
    it('should update template', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await TicketApi.updateTemplate(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/templates/1', { name: 'Updated' });
    });
  });

  describe('deleteTemplate', () => {
    it('should delete template', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketApi.deleteTemplate(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/templates/1');
    });
  });

  describe('updateTemplateStatus', () => {
    it('should update template status', async () => {
      mockPatch.mockResolvedValue({ id: 1 });
      await TicketApi.updateTemplateStatus(1, true);
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/tickets/templates/1/status', { isActive: true });
    });
  });

  describe('getTicketSLA', () => {
    it('should get ticket SLA', async () => {
      mockGet.mockResolvedValue({ ticketId: 1, isBreached: false });
      await TicketApi.getTicketSLA(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/sla');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors from httpClient', async () => {
      mockPost.mockRejectedValue(new Error('Network error'));
      await expect(TicketApi.approveTicket(1, { action: 'approve', ticketId: 1 })).rejects.toThrow('Network error');
    });
  });
});
