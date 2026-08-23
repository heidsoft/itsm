import { dashboardService, ticketAnalyticsService } from '../analytics-service';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockRequest = (httpClient as any).request as jest.Mock;

describe('DashboardService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getOverview', () => {
    it('should fetch dashboard overview', async () => {
      const overview = {
        overview: { totalTickets: 100, pendingTickets: 20, inProgressTickets: 30, resolvedToday: 10, avgResponseTime: 5, avgResolutionTime: 60 },
        recentActivities: [],
        kpiMetrics: [],
        ticketTrend: [],
        slaData: { complianceRate: 0.95, responseTimeCompliance: 0.9, resolutionTimeCompliance: 0.85, atRiskTickets: 2, breachedTickets: 1 },
        satisfactionData: { averageRating: 4.2, totalRatings: 50, ratingDistribution: [] },
      };
      mockGet.mockResolvedValue(overview);

      const result = await dashboardService.getOverview();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/overview');
      expect(result.overview.totalTickets).toBe(100);
    });
  });

  describe('getKPIMetrics', () => {
    it('should fetch KPI metrics', async () => {
      const metrics = [{ id: 'kpi1', title: 'Response Time', value: 5, unit: 'min', color: 'green', trend: 'up', changePercent: 10 }];
      mockGet.mockResolvedValue(metrics);

      const result = await dashboardService.getKPIMetrics();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/kpi-metrics');
      expect(result).toHaveLength(1);
      expect(result[0].title).toBe('Response Time');
    });
  });

  describe('getTicketTrend', () => {
    it('should fetch ticket trend data', async () => {
      const trend = [{ date: '2024-01-01', created: 5, resolved: 3, pending: 2 }];
      mockGet.mockResolvedValue(trend);

      const result = await dashboardService.getTicketTrend();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/ticket-trend');
      expect(result[0].created).toBe(5);
    });
  });

  describe('getIncidentDistribution', () => {
    it('should fetch incident distribution', async () => {
      const distribution = { critical: 2, high: 5, medium: 10 };
      mockGet.mockResolvedValue(distribution);

      const result = await dashboardService.getIncidentDistribution();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/incident-distribution');
      expect(result).toEqual(distribution);
    });
  });

  describe('getSLAData', () => {
    it('should fetch SLA data', async () => {
      const slaData = { complianceRate: 0.95, responseTimeCompliance: 0.9, resolutionTimeCompliance: 0.85, atRiskTickets: 3, breachedTickets: 1 };
      mockGet.mockResolvedValue(slaData);

      const result = await dashboardService.getSLAData();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/sla-data');
      expect(result.complianceRate).toBe(0.95);
    });
  });

  describe('getSatisfactionData', () => {
    it('should fetch satisfaction data', async () => {
      const data = { averageRating: 4.5, totalRatings: 100, ratingDistribution: [{ rating: 5, count: 60 }] };
      mockGet.mockResolvedValue(data);

      const result = await dashboardService.getSatisfactionData();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/satisfaction-data');
      expect(result.averageRating).toBe(4.5);
    });
  });

  describe('getQuickActions', () => {
    it('should fetch quick actions', async () => {
      const actions = [{ id: 1, label: 'Create Ticket', icon: 'plus' }];
      mockGet.mockResolvedValue(actions);

      const result = await dashboardService.getQuickActions();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/quick-actions');
      expect(result).toHaveLength(1);
    });
  });

  describe('getRecentActivities', () => {
    it('should fetch recent activities', async () => {
      const activities = [{ id: 1, type: 'ticket_created', description: 'New ticket', user: 'Alice', timestamp: '2024-01-01' }];
      mockGet.mockResolvedValue(activities);

      const result = await dashboardService.getRecentActivities();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/dashboard/recent-activities');
      expect(result[0].type).toBe('ticket_created');
    });
  });
});

describe('TicketAnalyticsService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getAnalytics', () => {
    it('should fetch analytics data', async () => {
      const raw = {
        total: 200,
        statusGroups: [
          { status: 'new', count: 80 },
          { status: 'in_progress', count: 50 },
          { status: 'resolved', count: 70 },
        ],
        priorityGroups: [
          { priority: 'high', count: 40 },
          { priority: 'medium', count: 100 },
          { priority: 'low', count: 60 },
        ],
        trend30d: [{ date: '2024-01-15', count: 10 }],
        generatedAt: '2024-01-31T00:00:00Z',
      };
      mockGet.mockResolvedValue(raw);

      const result = await ticketAnalyticsService.getAnalytics({ dateFrom: '2024-01-01', dateTo: '2024-01-31' });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/analytics/tickets', { dateFrom: '2024-01-01', dateTo: '2024-01-31' });
      expect(result.totalTickets).toBe(200);
      expect(result.priorityDistribution).toHaveLength(3);
      expect(result.dailyTrend).toHaveLength(1);
    });

    it('should fetch analytics without params', async () => {
      mockGet.mockResolvedValue({ total: 0, statusGroups: [], priorityGroups: [], trend30d: [] });

      await ticketAnalyticsService.getAnalytics();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/analytics/tickets', {});
    });

    it('should gracefully handle empty response', async () => {
      mockGet.mockResolvedValue({});

      const result = await ticketAnalyticsService.getAnalytics();

      expect(result.totalTickets).toBe(0);
      expect(result.dailyTrend).toEqual([]);
      expect(result.priorityDistribution).toEqual([]);
    });
  });

  describe('getStats', () => {
    it('should fetch ticket stats', async () => {
      const stats = { total: 100, open: 20, inProgress: 30, pending: 10, resolved: 25, closed: 15 };
      mockGet.mockResolvedValue(stats);

      const result = await ticketAnalyticsService.getStats();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/stats');
      expect(result.total).toBe(100);
      expect(result.open).toBe(20);
    });
  });

  describe('exportAnalytics', () => {
    it('should export analytics as blob', async () => {
      const blob = new Blob(['data'], { type: 'text/csv' });
      mockRequest.mockResolvedValue(blob);

      const result = await ticketAnalyticsService.exportAnalytics({
        dateFrom: '2024-01-01',
        dateTo: '2024-01-31',
        format: 'csv',
      });

      expect(mockRequest).toHaveBeenCalledWith({
        method: 'GET',
        url: '/api/v1/tickets/analytics/export',
        params: { dateFrom: '2024-01-01', dateTo: '2024-01-31', format: 'csv' },
        responseType: 'blob',
      });
      expect(result).toBe(blob);
    });

    it('should export in excel format', async () => {
      const blob = new Blob(['excel-data']);
      mockRequest.mockResolvedValue(blob);

      await ticketAnalyticsService.exportAnalytics({
        dateFrom: '2024-01-01',
        dateTo: '2024-12-31',
        format: 'excel',
        groupBy: 'month',
      });

      expect(mockRequest).toHaveBeenCalledWith({
        method: 'GET',
        url: '/api/v1/tickets/analytics/export',
        params: { dateFrom: '2024-01-01', dateTo: '2024-12-31', format: 'excel', groupBy: 'month' },
        responseType: 'blob',
      });
    });
  });
});
