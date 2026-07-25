import { TicketAnalyticsApi } from '@/lib/api/ticket-analytics-api';
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

const mockPost = httpClient.post as jest.Mock;

describe('TicketAnalyticsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getDeepAnalytics', () => {
    it('should post analytics config', async () => {
      const config = {
        dimensions: ['status'],
        metrics: ['count'],
        chartType: 'bar' as const,
        timeRange: ['2024-01-01', '2024-12-31'] as [string, string],
        filters: {},
      };
      mockPost.mockResolvedValue({ data: [], summary: {}, generatedAt: '' });
      const result = await TicketAnalyticsApi.getDeepAnalytics(config);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/analytics/deep', config);
      expect(result).toHaveProperty('data');
    });
  });

  describe('exportAnalytics', () => {
    it('should export analytics as excel', async () => {
      const config = {
        dimensions: ['status'],
        metrics: ['count'],
        chartType: 'bar' as const,
        timeRange: ['2024-01-01', '2024-12-31'] as [string, string],
        filters: {},
      };
      const mockBlob = new Blob(['data']);
      mockPost.mockResolvedValue(mockBlob);
      const result = await TicketAnalyticsApi.exportAnalytics(config, 'excel');
      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/tickets/analytics/export?format=excel',
        config,
        { responseType: 'blob' }
      );
      expect(result).toBeInstanceOf(Blob);
    });
  });
});
