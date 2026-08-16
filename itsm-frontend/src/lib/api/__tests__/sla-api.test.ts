import { SLAApi } from '../sla-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
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

describe('SLAApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getSLADefinitions', () => {
    it('should get definitions with params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      await SLAApi.getSLADefinitions({ page: 1, size: 10, isActive: true, name: 'test' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions', { page: '1', size: '10', isActive: 'true', name: 'test' });
    });

    it('should get definitions without params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await SLAApi.getSLADefinitions();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions', {});
    });
  });

  describe('getSLADefinition', () => {
    it('should get definition by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Gold SLA' });
      const result = await SLAApi.getSLADefinition(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions/1');
      expect(result.name).toBe('Gold SLA');
    });
  });

  describe('createSLADefinition', () => {
    it('should create definition', async () => {
      const data = { name: 'New SLA', priority: 'high', responseTime: 60, resolutionTime: 240 };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await SLAApi.createSLADefinition(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/definitions', data);
      expect(result.name).toBe('New SLA');
    });
  });

  describe('updateSLADefinition', () => {
    it('should update definition', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      await SLAApi.updateSLADefinition(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/sla/definitions/1', { name: 'Updated' });
    });
  });

  describe('deleteSLADefinition', () => {
    it('should delete definition', async () => {
      mockDelete.mockResolvedValue(undefined);
      await SLAApi.deleteSLADefinition(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/sla/definitions/1');
    });
  });

  describe('checkTicketCompliance', () => {
    it('should check compliance', async () => {
      mockPost.mockResolvedValue({ isCompliant: true, violations: [] });
      const result = await SLAApi.checkTicketCompliance(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/check-compliance/1');
      expect(result.isCompliant).toBe(true);
    });
  });

  describe('getSLAViolations', () => {
    it('should get violations with params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await SLAApi.getSLAViolations({ page: 1, size: 10, severity: 'high' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/violations', expect.objectContaining({ page: '1', size: '10', severity: 'high' }));
    });
  });

  describe('updateSLAViolationStatus', () => {
    it('should update violation status', async () => {
      mockPut.mockResolvedValue(undefined);
      await SLAApi.updateSLAViolationStatus(1, true, 'resolved');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/sla/violations/1', { isResolved: true, notes: 'resolved' });
    });
  });

  describe('getSLAComplianceReport', () => {
    it('should get compliance report and transform data', async () => {
      mockGet.mockResolvedValue({ totalTickets: 100, metSla: 90, violatedSla: 10, complianceRate: 90, avgResponseTime: 30, avgResolutionTime: 120, reportPeriod: { startDate: '2024-01-01', endDate: '2024-01-31' } });
      const result = await SLAApi.getSLAComplianceReport({ startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/compliance-report', { start_date: '2024-01-01', end_date: '2024-01-31' });
      expect(result.totalTickets).toBe(100);
      expect(result.complianceRate).toBe(90);
    });

    it('should handle missing fields with defaults', async () => {
      mockGet.mockResolvedValue({});
      const result = await SLAApi.getSLAComplianceReport({ startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(result.totalTickets).toBe(0);
      expect(result.reportPeriod.startDate).toBe('2024-01-01');
    });
  });

  describe('getSLAMonitoring', () => {
    it('should transform monitoring data', async () => {
      mockPost.mockResolvedValue({ totalViolations: 10, resolvedViolations: 8, activeViolations: 2, complianceRate: 0.8, activeSlas: 3, activeAlertRules: 1 });
      const result = await SLAApi.getSLAMonitoring({ startTime: '30d', endTime: 'now' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/monitor', { startTime: '30d', endTime: 'now' });
      expect(result.complianceRate).toBe(80);
      expect(result.violationRate).toBe(20);
      expect(result.totalTickets).toBe(10);
    });
  });

  describe('getSLAMetrics', () => {
    it('should get metrics from monitor endpoint', async () => {
      mockPost.mockResolvedValue({ averageResponseTime: 30, averageResolutionTime: 120, complianceRate: 95, violatedTickets: 5 });
      const result = await SLAApi.getSLAMetrics({ period: 'month' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/monitor', {});
      expect(result.responseTimeAvg).toBe(30);
      expect(result.violationCount).toBe(5);
    });
  });

  describe('getSLAAlerts', () => {
    it('should get alerts from monitor', async () => {
      mockPost.mockResolvedValue({ alerts: [{ ticketId: 1, ticketTitle: 'Test', priority: 'high', slaDefinition: 'Gold', timeRemaining: 30, alertLevel: 'warning', createdAt: '2024-01-01' }] });
      const result = await SLAApi.getSLAAlerts();
      expect(result).toHaveLength(1);
      expect(result[0].ticketId).toBe(1);
    });

    it('should return empty array when no alerts', async () => {
      mockPost.mockResolvedValue({});
      const result = await SLAApi.getSLAAlerts();
      expect(result).toEqual([]);
    });
  });

  describe('getSLAStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalDefinitions: 5, activeDefinitions: 3 });
      await SLAApi.getSLAStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/stats');
    });
  });

  describe('triggerSLAMonitoring', () => {
    it('should trigger monitoring', async () => {
      mockPost.mockResolvedValue({ checkedTickets: 10, violationsFound: 2, alertsSent: 1 });
      await SLAApi.triggerSLAMonitoring();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/monitor');
    });
  });

  describe('createAlertRule', () => {
    it('should create alert rule', async () => {
      const data = { name: 'Rule1', slaDefinitionId: 1, alertLevel: 'warning' as const, thresholdPercentage: 80, notificationChannels: ['email'], isActive: true };
      mockPost.mockResolvedValue({ id: 1 });
      await SLAApi.createAlertRule(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/alert-rules', data);
    });
  });

  describe('updateAlertRule', () => {
    it('should update alert rule', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await SLAApi.updateAlertRule(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/sla/alert-rules/1', { name: 'Updated' });
    });
  });

  describe('deleteAlertRule', () => {
    it('should delete alert rule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await SLAApi.deleteAlertRule(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/sla/alert-rules/1');
    });
  });

  describe('getAlertRules', () => {
    it('should get alert rules', async () => {
      mockGet.mockResolvedValue([]);
      await SLAApi.getAlertRules({ slaDefinitionId: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/alert-rules', { slaDefinitionId: 1 });
    });
  });

  describe('getAlertRule', () => {
    it('should get alert rule by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Rule1' });
      await SLAApi.getAlertRule(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/alert-rules/1');
    });
  });

  describe('getAlertHistory', () => {
    it('should get alert history', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      await SLAApi.getAlertHistory({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/alert-history', { page: 1, pageSize: 10 });
    });
  });

  describe('deprecated aliases', () => {
    it('getDefinitions should call getSLADefinitions', async () => {
      mockGet.mockResolvedValue({ items: [] });
      await SLAApi.getDefinitions({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions', expect.any(Object));
    });

    it('getDefinition should call getSLADefinition', async () => {
      mockGet.mockResolvedValue({ id: 1 });
      await SLAApi.getDefinition(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/definitions/1');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Server error'));
      await expect(SLAApi.getSLADefinition(999)).rejects.toThrow('Server error');
    });
  });
});
