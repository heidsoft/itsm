import { TicketPredictionApi } from '@/lib/api/ticket-prediction-api';
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

describe('TicketPredictionApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTrendPrediction', () => {
    it('should post prediction request', async () => {
      const params = {
        timeRange: ['2024-01-01', '2024-06-30'] as [string, string],
        predictionPeriod: 'month' as const,
        modelType: 'arima' as const,
      };
      mockPost.mockResolvedValue({ period: 'month', summary: 'test', data: [], metrics: {}, generatedAt: '' });
      const result = await TicketPredictionApi.getTrendPrediction(params);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/prediction/trend', params);
      expect(result).toHaveProperty('period');
    });
  });

  describe('exportPredictionReport', () => {
    it('should export prediction as excel', async () => {
      const params = {
        timeRange: ['2024-01-01', '2024-06-30'] as [string, string],
        predictionPeriod: 'month' as const,
        modelType: 'arima' as const,
      };
      const mockBlob = new Blob(['data']);
      mockPost.mockResolvedValue(mockBlob);
      const result = await TicketPredictionApi.exportPredictionReport(params, 'excel');
      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/tickets/prediction/export?format=excel',
        params,
        { responseType: 'blob' }
      );
      expect(result).toBeInstanceOf(Blob);
    });
  });
});
