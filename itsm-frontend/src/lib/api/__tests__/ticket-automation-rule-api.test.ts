import { TicketAutomationRuleApi } from '@/lib/api/ticket-automation-rule-api';
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

describe('TicketAutomationRuleApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('listRules', () => {
    it('should list automation rules', async () => {
      mockGet.mockResolvedValue({ rules: [{ id: 1, name: 'Auto close' }], total: 1 });
      const result = await TicketAutomationRuleApi.listRules();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/automation-rules');
      expect(result.rules).toHaveLength(1);
    });
  });

  describe('getRule', () => {
    it('should get a rule by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Auto close' });
      const result = await TicketAutomationRuleApi.getRule(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/automation-rules/1');
      expect(result.id).toBe(1);
    });
  });

  describe('createRule', () => {
    it('should create a rule', async () => {
      const data = { name: 'New Rule', priority: 1, conditions: [], actions: [{ type: 'notify' }], isActive: true };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await TicketAutomationRuleApi.createRule(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/automation-rules', data);
      expect(result.id).toBe(2);
    });
  });

  describe('updateRule', () => {
    it('should update a rule', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated Rule' });
      const result = await TicketAutomationRuleApi.updateRule(1, { name: 'Updated Rule' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/automation-rules/1', { name: 'Updated Rule' });
      expect(result.name).toBe('Updated Rule');
    });
  });

  describe('deleteRule', () => {
    it('should delete a rule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketAutomationRuleApi.deleteRule(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/automation-rules/1');
    });
  });

  describe('testRule', () => {
    it('should test a rule against a ticket', async () => {
      mockPost.mockResolvedValue({ matched: true, actions: ['notify'], reason: 'conditions met' });
      const result = await TicketAutomationRuleApi.testRule({ ruleId: 1, ticketId: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/automation-rules/1/test', { ticketId: 5 });
      expect(result.matched).toBe(true);
    });
  });
});
