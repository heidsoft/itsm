import { TicketAnalyticsApi } from '@/lib/api/ticket-analytics-api';
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

const mockPost = httpClient.post as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

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
    const config = {
      dimensions: ['status'],
      metrics: ['count'],
      chartType: 'bar' as const,
      timeRange: ['2024-01-01', '2024-12-31'] as [string, string],
      filters: {},
    };

    it('should post the config as body and keep format in the query', async () => {
      const mockBlob = new Blob(['data'], { type: 'application/vnd.ms-excel' });
      mockRequest.mockResolvedValue(mockBlob);

      const result = await TicketAnalyticsApi.exportAnalytics(config, 'excel');

      expect(mockRequest).toHaveBeenCalledWith({
        method: 'POST',
        url: '/api/v1/tickets/analytics/export',
        params: { format: 'excel' },
        data: config,
        responseType: 'blob',
      });
      expect(result).toBe(mockBlob);
    });

    it('should let the server error message propagate to the caller', async () => {
      mockRequest.mockRejectedValue(new Error('timeRange 必须恰好包含起止两个日期（YYYY-MM-DD）'));

      await expect(TicketAnalyticsApi.exportAnalytics(config, 'csv')).rejects.toThrow(
        'timeRange 必须恰好包含起止两个日期'
      );
    });
  });
});
