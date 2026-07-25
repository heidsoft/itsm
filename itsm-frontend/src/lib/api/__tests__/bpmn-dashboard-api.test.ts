import { BPMNDashboardApi } from '@/lib/api/bpmn-dashboard-api';
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

describe('BPMNDashboardApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getDashboardMetrics', () => {
    it('should get metrics with tenantId', async () => {
      const expected = { totalProcesses: 10, activeInstances: 5 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getDashboardMetrics(1);
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/dashboard/metrics?tenantId=1'));
      expect(res).toEqual(expected);
    });

    it('should include time params', async () => {
      mockGet.mockResolvedValue({});
      await BPMNDashboardApi.getDashboardMetrics(1, '2024-01-01', '2024-01-31');
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('start_time=2024-01-01'));
    });
  });

  describe('getProcessMetrics', () => {
    it('should get process metrics by key', async () => {
      const expected = { totalInstances: 100 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getProcessMetrics('proc1', 1);
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/dashboard/process/proc1/metrics'));
      expect(res).toEqual(expected);
    });
  });

  describe('queryAuditLogs', () => {
    it('should query audit logs with params', async () => {
      const expected = { list: [], total: 0, page: 1 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.queryAuditLogs({ tenantId: 1, page: 1 });
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/dashboard/audit-logs'));
      expect(res).toEqual(expected);
    });
  });

  describe('getProcessTimeline', () => {
    it('should get timeline entries (array response)', async () => {
      const entries = [{ id: 1, action: 'start' }];
      mockGet.mockResolvedValue(entries);
      const res = await BPMNDashboardApi.getProcessTimeline('inst1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/instances/inst1/timeline');
      expect(res).toEqual(entries);
    });

    it('should get timeline entries (wrapped response)', async () => {
      const entries = [{ id: 1, action: 'start' }];
      mockGet.mockResolvedValue({ data: { entries } });
      const res = await BPMNDashboardApi.getProcessTimeline('inst1');
      expect(res).toEqual(entries);
    });
  });

  describe('getUserActivity', () => {
    it('should get user activity', async () => {
      const expected = [{ id: 1 }];
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getUserActivity(5, 1);
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/dashboard/audit-logs/user/5'));
      expect(res).toEqual(expected);
    });
  });

  describe('getSLAViolations', () => {
    it('should get SLA violations', async () => {
      const expected = [{ resourceType: 'ticket' }];
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getSLAViolations(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/dashboard/sla/violations?tenant_id=1');
      expect(res).toEqual(expected);
    });
  });

  describe('getSLACompliance', () => {
    it('should get SLA compliance', async () => {
      const expected = { complianceRate: 95, compliant: 19, total: 20 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getSLACompliance('proc1', 1);
      expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('/api/v1/bpmn/dashboard/sla/compliance'));
      expect(res).toEqual(expected);
    });
  });

  describe('getTenantStats', () => {
    it('should get tenant stats', async () => {
      const expected = { totalDefinitions: 5 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getTenantStats(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/dashboard/tenant/stats?tenant_id=1');
      expect(res).toEqual(expected);
    });
  });

  describe('getBottleneckAnalysis', () => {
    it('should get bottleneck analysis', async () => {
      const expected = [{ taskName: 'Approval', totalCount: 10 }];
      mockGet.mockResolvedValue(expected);
      const res = await BPMNDashboardApi.getBottleneckAnalysis('proc1', 1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/dashboard/bottlenecks?key=proc1&tenant_id=1');
      expect(res).toEqual(expected);
    });
  });
});
