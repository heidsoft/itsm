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
      // 契约：后端 dto.AutoAssignResponse.assignmentType 实测只发 auto/rule/ticket_type_rule/manual。
      mockPost.mockResolvedValue({ ticketId: 1, assignedTo: 5, assignmentType: 'rule', reason: 'matched rule', score: 80 });
      const result = await TicketAssignmentApi.autoAssign(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/auto-assign');
      expect(result.assignedTo).toBe(5);
      expect(result.assignmentType).toBe('rule');
    });
  });

  describe('getRecommendations', () => {
    it('should fetch assign recommendations as an items envelope', async () => {
      // 契约：后端 dto.AssignRecommendationListResponse 是不分页的 {items,total}。
      mockGet.mockResolvedValue({
        items: [{ userId: 1, username: 'john', name: 'John', email: 'j@e.com', score: 90, reason: 'skill', workload: 2 }],
        total: 1,
      });
      const result = await TicketAssignmentApi.getRecommendations(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assign-recommendations/10');
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(1);
      expect(result).not.toHaveProperty('recommendations');
    });
  });

  describe('listRules', () => {
    it('should list assignment rules', async () => {
      // 契约：后端 dto.ListAssignmentRulesResponse 是不分页的 {items,total}。
      mockGet.mockResolvedValue({ items: [{ id: 1, name: 'Rule1', conditions: [], actions: { type: 'user' }, isActive: true, executionCount: 0, createdAt: '', updatedAt: '' }], total: 1 });
      const result = await TicketAssignmentApi.listRules();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/assignment-rules');
      expect(result.items).toHaveLength(1);
      expect(result).not.toHaveProperty('rules');
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
