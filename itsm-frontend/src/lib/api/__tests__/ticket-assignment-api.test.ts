import { TicketAssignmentApi } from '@/lib/api/ticket-assignment-api';
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

describe('TicketAssignmentApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('autoAssign', () => {
    it('should auto-assign a ticket', async () => {
      mockPost.mockResolvedValue({ ticketId: 1, assignedTo: 5, assignmentType: 'smart', reason: 'best match' });
      const result = await TicketAssignmentApi.autoAssign(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/auto-assign');
      expect(result.assignedTo).toBe(5);
    });
  });

  describe('getRecommendations', () => {
    it('should fetch assign recommendations', async () => {
      mockGet.mockResolvedValue({ recommendations: [{ userId: 1, userName: 'John', score: 90, reason: 'skill', factors: {} }], total: 1 });
      const result = await TicketAssignmentApi.getRecommendations(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assign-recommendations/10');
      expect(result.recommendations).toHaveLength(1);
    });
  });

  describe('listRules', () => {
    it('should list assignment rules', async () => {
      mockGet.mockResolvedValue({ rules: [{ id: 1, name: 'Rule1', conditions: [], actions: { type: 'user' }, isActive: true, executionCount: 0, createdAt: '', updatedAt: '' }], total: 1 });
      const result = await TicketAssignmentApi.listRules();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules');
      expect(result.rules).toHaveLength(1);
    });
  });

  describe('getRule', () => {
    it('should get a rule by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Rule1', conditions: [], actions: { type: 'user' }, isActive: true, executionCount: 0, createdAt: '', updatedAt: '' });
      const result = await TicketAssignmentApi.getRule(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/1');
      expect(result.id).toBe(1);
    });
  });

  describe('createRule', () => {
    it('should create a rule', async () => {
      const data = { name: 'New', priority: 1, conditions: [], actions: { type: 'user' as const } };
      mockPost.mockResolvedValue({ id: 2, ...data, isActive: true, executionCount: 0, createdAt: '', updatedAt: '' });
      const result = await TicketAssignmentApi.createRule(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules', expect.any(Object));
      expect(result.id).toBe(2);
    });
  });

  describe('updateRule', () => {
    it('should update a rule', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated', conditions: [], actions: { type: 'user' }, isActive: true, executionCount: 0, createdAt: '', updatedAt: '' });
      const result = await TicketAssignmentApi.updateRule(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/1', expect.any(Object));
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteRule', () => {
    it('should delete a rule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketAssignmentApi.deleteRule(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/1');
    });
  });

  describe('testRule', () => {
    it('should test a rule', async () => {
      mockPost.mockResolvedValue({ matched: true, assignedTo: 5, reason: 'matched' });
      const result = await TicketAssignmentApi.testRule({ ruleId: 1, ticketId: 10 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules/test', expect.any(Object));
      expect(result.matched).toBe(true);
    });
  });
});
