/**
 * Tests for ticket-constants.ts
 *
 * 关键回归点：配置表的键必须与后端工单词表逐字一致（snake_case）。
 * 旧实现用 inProgress/pendingApproval 等 camelCase 键，导致
 * getStatusConfig('in_progress') 静默落到 open 兜底，徽标长期错显示。
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { TicketStatus } from '@/constants/taxonomy';
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
    it('covers every status in the backend ticket state machine', () => {
      Object.values(TicketStatus).forEach((status) => {
        expect(TICKET_STATUS_CONFIG[status]).toBeDefined();
      });
    });

    it('keys the config with the backend snake_case vocabulary', () => {
      expect(Object.keys(TICKET_STATUS_CONFIG).sort()).toEqual(
        [
          'new',
          'open',
          'assigned',
          'in_progress',
          'pending',
          'resolved',
          'closed',
          'cancelled',
          'approved',
          'rejected',
        ].sort()
      );
    });

    it('does not treat pending_approval as a ticket status', () => {
      expect(TICKET_STATUS_CONFIG['pending_approval' as TicketStatus]).toBeUndefined();
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
      expect(TICKET_TYPE_CONFIG.service_request.text).toBe('服务请求');
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

    it('resolves snake_case statuses instead of falling back to open', () => {
      expect(getStatusConfig('in_progress')).toBe(TICKET_STATUS_CONFIG.in_progress);
      expect(getStatusConfig('assigned')).toBe(TICKET_STATUS_CONFIG.assigned);
      expect(getStatusConfig('approved')).toBe(TICKET_STATUS_CONFIG.approved);
      expect(getStatusConfig('rejected')).toBe(TICKET_STATUS_CONFIG.rejected);
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
    });

    it('should return service request config for unknown type', () => {
      expect(getTypeConfig('unknown')).toBe(TICKET_TYPE_CONFIG.service_request);
    });
  });
});
