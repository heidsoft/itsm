/**
 * SLAService unit tests
 */
import { slaService, SLAStatus, SLAType, SLAPriority, EscalationLevel } from '../sla-service';
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
const mockDelete = httpClient.delete as jest.Mock;
const mockPut = httpClient.put as jest.Mock;

describe('SLAService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getSLADefinitions', () => {
    it('should call GET /api/v1/sla/definitions without query when no params', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await slaService.getSLADefinitions();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions');
    });

    it('should build query string from params', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await slaService.getSLADefinitions({ status: SLAStatus.ACTIVE, page: 1, pageSize: 10 });
      const url = (mockGet as jest.Mock).mock.calls[0][0];
      expect(url).toContain('/api/v1/sla/definitions?');
      expect(url).toContain('status=active');
      expect(url).toContain('page=1');
      expect(url).toContain('pageSize=10');
    });

    it('should skip undefined params', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await slaService.getSLADefinitions({ status: SLAStatus.ACTIVE, type: undefined });
      const url = (mockGet as jest.Mock).mock.calls[0][0];
      expect(url).not.toContain('type=');
    });
  });

  describe('getSLADefinition', () => {
    it('should call GET /api/v1/sla/definitions/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 1, name: 'Gold SLA' });
      const result = await slaService.getSLADefinition(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions/1');
      expect(result.name).toBe('Gold SLA');
    });
  });

  describe('createSLADefinition', () => {
    it('should call POST /api/v1/sla/definitions', async () => {
      const data = {
        name: 'Gold',
        type: SLAType.RESPONSE_TIME,
        priority: SLAPriority.HIGH,
        targetTime: 60,
        warningTime: 45,
        businessHoursId: 1,
        escalationRules: [],
        applicableTo: { ticketTypes: [], categories: [], priorities: [], departments: [] },
      };
      mockPost.mockResolvedValueOnce({ id: 5, ...data });
      const result = await slaService.createSLADefinition(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/definitions', data);
      expect(result.id).toBe(5);
    });
  });

  describe('updateSLADefinition', () => {
    it('should call PUT /api/v1/sla/definitions/:id', async () => {
      mockPut.mockResolvedValueOnce({ id: 1, name: 'Updated' });
      const result = await slaService.updateSLADefinition(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/sla/definitions/1', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteSLADefinition', () => {
    it('should call DELETE /api/v1/sla/definitions/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await slaService.deleteSLADefinition(3);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/sla/definitions/3');
    });
  });

  describe('getSLAInstances', () => {
    it('should call GET /api/v1/sla/instances without query when no params', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await slaService.getSLAInstances();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/instances');
    });

    it('should build query string from params', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await slaService.getSLAInstances({ status: 'warning', page: 2 });
      const url = (mockGet as jest.Mock).mock.calls[0][0];
      expect(url).toContain('status=warning');
      expect(url).toContain('page=2');
    });
  });

  describe('getSLAInstance', () => {
    it('should call GET /api/v1/sla/instances/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 10 });
      await slaService.getSLAInstance(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/instances/10');
    });
  });

  describe('suspendSLAInstance', () => {
    it('should call POST /api/v1/sla/instances/:id/suspend', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await slaService.suspendSLAInstance(5, 'Waiting for vendor');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/instances/5/suspend', { reason: 'Waiting for vendor' });
    });
  });

  describe('resumeSLAInstance', () => {
    it('should call POST /api/v1/sla/instances/:id/resume', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await slaService.resumeSLAInstance(5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/instances/5/resume');
    });
  });

  describe('getSLAStats', () => {
    it('should call GET /api/v1/sla/stats without query when no params', async () => {
      mockGet.mockResolvedValueOnce({ totalInstances: 100 });
      await slaService.getSLAStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/stats');
    });

    it('should build query string from params', async () => {
      mockGet.mockResolvedValueOnce({ totalInstances: 50 });
      await slaService.getSLAStats({ dateRange: '30d' });
      const url = (mockGet as jest.Mock).mock.calls[0][0];
      expect(url).toContain('dateRange=30d');
    });
  });

  describe('getSLAReport', () => {
    it('should build query and call GET /api/v1/sla/reports', async () => {
      mockGet.mockResolvedValueOnce({ period: { start: '2024-01-01', end: '2024-01-31' } });
      await slaService.getSLAReport({ startDate: '2024-01-01', endDate: '2024-01-31' });
      const url = (mockGet as jest.Mock).mock.calls[0][0];
      expect(url).toContain('/api/v1/sla/reports?');
      expect(url).toContain('startDate=2024-01-01');
      expect(url).toContain('endDate=2024-01-31');
    });
  });

  describe('getBusinessHours', () => {
    it('should call GET /api/v1/sla/business-hours', async () => {
      mockGet.mockResolvedValueOnce([{ id: 1, name: 'Default' }]);
      const result = await slaService.getBusinessHours();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/business-hours');
      expect(result).toHaveLength(1);
    });
  });

  describe('createBusinessHours', () => {
    it('should call POST /api/v1/sla/business-hours', async () => {
      const data = { name: 'Custom', timezone: 'Asia/Shanghai', workingDays: [1, 2, 3, 4, 5], workingHours: { start: '09:00', end: '18:00' }, isDefault: false, description: '' };
      mockPost.mockResolvedValueOnce({ id: 2, ...data });
      await slaService.createBusinessHours(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/business-hours', data);
    });
  });

  describe('getHolidays', () => {
    it('should call GET /api/v1/sla/holidays', async () => {
      mockGet.mockResolvedValueOnce([]);
      await slaService.getHolidays();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/holidays');
    });
  });

  describe('createHoliday', () => {
    it('should call POST /api/v1/sla/holidays', async () => {
      const data = { name: 'National Day', date: '2024-10-01', type: 'national' as const, isRecurring: true };
      mockPost.mockResolvedValueOnce({ id: 1, ...data });
      await slaService.createHoliday(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/holidays', data);
    });
  });

  describe('calculateBusinessTime', () => {
    it('should return 0 (placeholder implementation)', () => {
      const bh = { id: 1, name: 'Default', timezone: 'UTC', workingDays: [1, 2, 3, 4, 5], workingHours: { start: '09:00', end: '18:00' }, isDefault: true, createdAt: '', updatedAt: '' };
      const result = slaService.calculateBusinessTime('2024-01-01T09:00:00Z', '2024-01-01T17:00:00Z', bh, []);
      expect(result).toBe(0);
    });
  });

  describe('calculateSLAStatus', () => {
    it('should return resolved if instance is resolved', () => {
      const instance = { status: 'resolved', warningTime: '2020-01-01', breachTime: '2020-01-02' } as never;
      expect(slaService.calculateSLAStatus(instance)).toBe('resolved');
    });

    it('should return breached if now >= breachTime', () => {
      const instance = { status: 'active', warningTime: '2020-01-01', breachTime: '2020-01-02' } as never;
      expect(slaService.calculateSLAStatus(instance)).toBe('breached');
    });

    it('should return warning if now >= warningTime but < breachTime', () => {
      const future = new Date(Date.now() + 86400000).toISOString();
      const instance = { status: 'active', warningTime: '2020-01-01', breachTime: future } as never;
      expect(slaService.calculateSLAStatus(instance)).toBe('warning');
    });

    it('should return active if before warningTime', () => {
      const future1 = new Date(Date.now() + 86400000).toISOString();
      const future2 = new Date(Date.now() + 172800000).toISOString();
      const instance = { status: 'active', warningTime: future1, breachTime: future2 } as never;
      expect(slaService.calculateSLAStatus(instance)).toBe('active');
    });
  });

  describe('checkSLAWarnings', () => {
    it('should call GET /api/v1/sla/warnings', async () => {
      mockGet.mockResolvedValueOnce([]);
      await slaService.checkSLAWarnings();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/warnings');
    });
  });

  describe('processEscalation', () => {
    it('should call POST /api/v1/sla/instances/:id/escalate', async () => {
      mockPost.mockResolvedValueOnce(undefined);
      await slaService.processEscalation(7, EscalationLevel.LEVEL_2);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/instances/7/escalate', { level: 'level_2' });
    });
  });
});
