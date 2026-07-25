/**
 * Tests for ticket-constants.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import {
  TICKET_STATUS_CONFIG,
  TICKET_PRIORITY_CONFIG,
  TICKET_TYPE_CONFIG,
  getStatusConfig,
  getPriorityConfig,
  getTypeConfig,
} from '../ticket-constants';

describe('Ticket Constants', () => {
  describe('TICKET_STATUS_CONFIG', () => {
    it('should have all status keys', () => {
      expect(TICKET_STATUS_CONFIG.new).toBeDefined();
      expect(TICKET_STATUS_CONFIG.open).toBeDefined();
      expect(TICKET_STATUS_CONFIG.inProgress).toBeDefined();
      expect(TICKET_STATUS_CONFIG.pendingApproval).toBeDefined();
      expect(TICKET_STATUS_CONFIG.resolved).toBeDefined();
      expect(TICKET_STATUS_CONFIG.closed).toBeDefined();
      expect(TICKET_STATUS_CONFIG.cancelled).toBeDefined();
    });
  });

  describe('TICKET_PRIORITY_CONFIG', () => {
    it('should have all priority keys', () => {
      expect(TICKET_PRIORITY_CONFIG.low).toBeDefined();
      expect(TICKET_PRIORITY_CONFIG.medium).toBeDefined();
      expect(TICKET_PRIORITY_CONFIG.high).toBeDefined();
      expect(TICKET_PRIORITY_CONFIG.urgent).toBeDefined();
      expect(TICKET_PRIORITY_CONFIG.critical).toBeDefined();
    });
  });

  describe('TICKET_TYPE_CONFIG', () => {
    it('should have all type keys with text field', () => {
      expect(TICKET_TYPE_CONFIG.incident.text).toBe('事件');
      expect(TICKET_TYPE_CONFIG.serviceRequest.text).toBe('服务请求');
      expect(TICKET_TYPE_CONFIG.problem.text).toBe('问题');
      expect(TICKET_TYPE_CONFIG.change.text).toBe('变更');
    });
  });

  describe('getStatusConfig', () => {
    it('should return config for valid status', () => {
      expect(getStatusConfig('new')).toBe(TICKET_STATUS_CONFIG.new);
      expect(getStatusConfig('open')).toBe(TICKET_STATUS_CONFIG.open);
      expect(getStatusConfig('resolved')).toBe(TICKET_STATUS_CONFIG.resolved);
    });

    it('should return open config for unknown status', () => {
      expect(getStatusConfig('unknown')).toBe(TICKET_STATUS_CONFIG.open);
    });
  });

  describe('getPriorityConfig', () => {
    it('should return config for valid priority', () => {
      expect(getPriorityConfig('low')).toBe(TICKET_PRIORITY_CONFIG.low);
      expect(getPriorityConfig('high')).toBe(TICKET_PRIORITY_CONFIG.high);
      expect(getPriorityConfig('critical')).toBe(TICKET_PRIORITY_CONFIG.critical);
    });

    it('should return medium config for unknown priority', () => {
      expect(getPriorityConfig('unknown')).toBe(TICKET_PRIORITY_CONFIG.medium);
    });
  });

  describe('getTypeConfig', () => {
    it('should return config for valid type', () => {
      expect(getTypeConfig('incident')).toBe(TICKET_TYPE_CONFIG.incident);
      expect(getTypeConfig('problem')).toBe(TICKET_TYPE_CONFIG.problem);
    });

    it('should return serviceRequest config for unknown type', () => {
      expect(getTypeConfig('unknown')).toBe(TICKET_TYPE_CONFIG.serviceRequest);
    });
  });
});
