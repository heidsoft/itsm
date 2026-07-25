import { DashboardAPI } from '@/lib/api/dashboard-api';
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

describe('DashboardAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getOverview', () => {
    it('should get overview', async () => {
      const expected = { kpiMetrics: [] };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getOverview();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/overview');
      expect(res).toEqual(expected);
    });
  });

  describe('getKPIMetrics', () => {
    it('should get KPI metrics', async () => {
      const expected = [{ label: 'Total', value: 100 }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getKPIMetrics();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/kpi-metrics');
      expect(res).toEqual(expected);
    });
  });

  describe('getTicketTrend', () => {
    it('should get ticket trend with days', async () => {
      const expected = [{ date: '2024-01-01', count: 10 }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getTicketTrend(14);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/ticket-trend', { days: 14 });
      expect(res).toEqual(expected);
    });
  });

  describe('getIncidentDistribution', () => {
    it('should get incident distribution', async () => {
      const expected = [{ category: 'Network', count: 5 }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getIncidentDistribution();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/incident-distribution');
      expect(res).toEqual(expected);
    });
  });

  describe('getSLAData', () => {
    it('should get SLA data', async () => {
      const expected = [{ name: 'P1', compliance: 95 }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getSLAData();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/sla-data');
      expect(res).toEqual(expected);
    });
  });

  describe('getSatisfactionData', () => {
    it('should get satisfaction data', async () => {
      const expected = [{ month: 'Jan', score: 4.5 }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getSatisfactionData(6);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/satisfaction-data', { months: 6 });
      expect(res).toEqual(expected);
    });
  });

  describe('getQuickActions', () => {
    it('should get quick actions', async () => {
      const expected = [{ id: '1', label: 'New Ticket' }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getQuickActions();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/quick-actions');
      expect(res).toEqual(expected);
    });
  });

  describe('getRecentActivities', () => {
    it('should get recent activities', async () => {
      const expected = [{ id: '1', description: 'Created ticket' }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getRecentActivities(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/recent-activities', { limit: 5 });
      expect(res).toEqual(expected);
    });
  });

  describe('getDashboardConfig', () => {
    it('should get config', async () => {
      const expected = { id: '1', widgets: [] };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getDashboardConfig(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/config', { userId: 1 });
      expect(res).toEqual(expected);
    });
  });

  describe('saveDashboardConfig', () => {
    it('should save config', async () => {
      const config = { id: '1', widgets: [] } as any;
      mockPost.mockResolvedValue({ success: true });
      const res = await DashboardAPI.saveDashboardConfig(config);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/config', config);
      expect(res).toEqual({ success: true });
    });
  });

  describe('getDashboardLayout', () => {
    it('should get layout', async () => {
      const expected = { columns: 3 };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getDashboardLayout();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/layout', {});
      expect(res).toEqual(expected);
    });
  });

  describe('saveDashboardLayout', () => {
    it('should save layout', async () => {
      const layout = { columns: 4 } as any;
      mockPost.mockResolvedValue({ success: true });
      const res = await DashboardAPI.saveDashboardLayout(layout);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/layout', layout);
      expect(res).toEqual({ success: true });
    });
  });

  describe('getTicketStats', () => {
    it('should get ticket stats', async () => {
      const expected = { total: 100, open: 20 };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getTicketStats({ priority: 'high' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/stats/tickets', { priority: 'high' });
      expect(res).toEqual(expected);
    });
  });

  describe('getChartData', () => {
    it('should get chart data', async () => {
      const expected = { labels: [], datasets: [] };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getChartData('bar');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/charts/bar', undefined);
      expect(res).toEqual(expected);
    });
  });

  describe('getRealtimeData', () => {
    it('should get realtime data', async () => {
      const expected = { value: 42 };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getRealtimeData('activeUsers');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/realtime/activeUsers');
      expect(res).toEqual(expected);
    });
  });

  describe('getWidgetData', () => {
    it('should get widget data', async () => {
      const expected = { id: 'w1', type: 'chart' };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getWidgetData('w1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/widgets/w1/data', undefined);
      expect(res).toEqual(expected);
    });
  });

  describe('refreshWidgetData', () => {
    it('should refresh widget', async () => {
      const expected = { id: 'w1', type: 'chart' };
      mockPost.mockResolvedValue(expected);
      const res = await DashboardAPI.refreshWidgetData('w1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/widgets/w1/refresh', undefined);
      expect(res).toEqual(expected);
    });
  });

  describe('addWidget', () => {
    it('should add widget', async () => {
      const config = { type: 'chart' };
      mockPost.mockResolvedValue({ widget: { id: 'w2', type: 'chart' } });
      const res = await DashboardAPI.addWidget(config as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/widgets', config);
      expect(res.widget).toBeDefined();
    });
  });

  describe('removeWidget', () => {
    it('should remove widget', async () => {
      mockDelete.mockResolvedValue({ success: true });
      const res = await DashboardAPI.removeWidget('w1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/dashboard/widgets/w1');
      expect(res).toEqual({ success: true });
    });
  });

  describe('generateReport', () => {
    it('should generate report', async () => {
      const expected = { id: 'r1', type: 'weekly' };
      mockPost.mockResolvedValue(expected);
      const res = await DashboardAPI.generateReport('weekly', { from: '2024-01-01' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/reports/weekly', { from: '2024-01-01' });
      expect(res).toEqual(expected);
    });
  });

  describe('getTemplates', () => {
    it('should get templates', async () => {
      const expected = [{ id: 't1', name: 'Default' }];
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/templates');
      expect(res).toEqual(expected);
    });
  });

  describe('applyTemplate', () => {
    it('should apply template', async () => {
      mockPost.mockResolvedValue({ success: true, config: {} });
      const res = await DashboardAPI.applyTemplate('t1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/templates/t1/apply');
      expect(res.success).toBe(true);
    });
  });

  describe('exportDashboard', () => {
    it('should export dashboard', async () => {
      mockPost.mockResolvedValue({ downloadUrl: 'https://example.com/file' });
      const res = await DashboardAPI.exportDashboard();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/dashboard/export', undefined);
      expect(res.downloadUrl).toBeDefined();
    });
  });

  describe('getPerformanceMetrics', () => {
    it('should get performance metrics', async () => {
      const expected = { loadTime: 100, renderTime: 50, dataFetchTime: 200, widgetCount: 5, memoryUsage: 1024 };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getPerformanceMetrics();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/metrics/performance');
      expect(res).toEqual(expected);
    });
  });

  describe('getUsageStats', () => {
    it('should get usage stats', async () => {
      const expected = { totalViews: 1000, uniqueUsers: 50, avgSessionDuration: 300, mostUsedWidgets: [], peakUsageHours: [] };
      mockGet.mockResolvedValue(expected);
      const res = await DashboardAPI.getUsageStats({ start: '2024-01-01', end: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/metrics/usage', { start: '2024-01-01', end: '2024-01-31' });
      expect(res).toEqual(expected);
    });
  });
});
