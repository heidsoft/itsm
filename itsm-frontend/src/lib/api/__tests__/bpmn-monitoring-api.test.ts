import { BPMNMonitoringApi } from '@/lib/api/bpmn-monitoring-api';
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

describe('BPMNMonitoringApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getProcessMetrics', () => {
    it('should get metrics with params', async () => {
      const expected = { totalInstances: 50, runningInstances: 10 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getProcessMetrics({ timeRange: '24h' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/metrics', { timeRange: '24h' });
      expect(res).toEqual(expected);
    });

    it('should handle empty params', async () => {
      const expected = { totalInstances: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await BPMNMonitoringApi.getProcessMetrics();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/metrics', {});
      expect(res).toEqual(expected);
    });
  });

  describe('getProcessMetricsByKey', () => {
    it('should get metrics by process key', async () => {
      const expected = { totalInstances: 20 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getProcessMetricsByKey('proc1', '7d');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/metrics/proc1', { time_range: '7d' });
      expect(res).toEqual(expected);
    });
  });

  describe('getProcessTimeline', () => {
    it('should get timeline for instance', async () => {
      const expected = { processInstanceId: 'inst1', entries: [], total: 0 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getProcessTimeline('inst1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/instances/inst1/timeline');
      expect(res).toEqual(expected);
    });
  });

  describe('getProcessInstanceStatus', () => {
    it('should get instance status', async () => {
      const expected = { instanceId: 'inst1', status: 'running', progress: 50 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getProcessInstanceStatus('inst1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/instances/inst1/status');
      expect(res).toEqual(expected);
    });
  });

  describe('listProcessInstancesStatus', () => {
    it('should list instances status with params', async () => {
      const expected = { instances: [], total: 0, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.listProcessInstancesStatus({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/instances/status', { page: '1', pageSize: '10' });
      expect(res).toEqual(expected);
    });
  });

  describe('getPerformanceMetrics', () => {
    it('should get performance metrics', async () => {
      const expected = { throughput: 100, avgLeadTime: 30 };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getPerformanceMetrics('24h');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/performance', { timeRange: '24h' });
      expect(res).toEqual(expected);
    });
  });

  describe('getPerformanceAlerts', () => {
    it('should return array response directly', async () => {
      const expected = [{ alertType: 'sla', severity: 'high', message: 'alert', timestamp: '2024-01-01' }];
      mockGet.mockResolvedValue(expected);
      const res = await BPMNMonitoringApi.getPerformanceAlerts();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/performance/alerts');
      expect(res).toEqual(expected);
    });

    it('should unwrap data field', async () => {
      const alerts = [{ alertType: 'sla', severity: 'high', message: 'alert', timestamp: '2024-01-01' }];
      mockGet.mockResolvedValue({ data: alerts });
      const res = await BPMNMonitoringApi.getPerformanceAlerts();
      expect(res).toEqual(alerts);
    });
  });

  describe('getSystemHealth', () => {
    it('should get system health', async () => {
      const expected = { status: 'healthy', components: {}, uptimeSeconds: 3600, version: '1.0' };
      mockGet.mockResolvedValue({ data: expected });
      const res = await BPMNMonitoringApi.getSystemHealth();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/health');
      expect(res).toEqual(expected);
    });
  });

  describe('getBottleneckAnalysis', () => {
    it('should extract bottleneckAnalysis from metrics', async () => {
      const bottleneck = { bottleneckTasks: [], slowestPaths: [], resourceConstraints: [], recommendations: [], severity: 'high' };
      mockGet.mockResolvedValue({ data: { totalInstances: 10, bottleneckAnalysis: bottleneck } });
      const res = await BPMNMonitoringApi.getBottleneckAnalysis('proc1', '24h');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/bpmn/monitoring/metrics/proc1', { timeRange: '24h' });
      expect(res).toEqual(bottleneck);
    });

    it('should return empty skeleton when bottleneckAnalysis is null', async () => {
      mockGet.mockResolvedValue({ data: { totalInstances: 10 } });
      const res = await BPMNMonitoringApi.getBottleneckAnalysis('proc1');
      expect(res).toEqual({
        bottleneckTasks: [],
        slowestPaths: [],
        resourceConstraints: [],
        recommendations: [],
        severity: 'low',
      });
    });
  });
});
