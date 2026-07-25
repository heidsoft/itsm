import { TicketRatingApi } from '@/lib/api/ticket-rating-api';
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

describe('TicketRatingApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('submitRating', () => {
    it('should submit a rating', async () => {
      const data = { rating: 5, comment: 'Great service' };
      mockPost.mockResolvedValue({ rating: 5, comment: 'Great service', ratedBy: 1 });
      const result = await TicketRatingApi.submitRating(10, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/10/rating', data);
      expect(result.rating).toBe(5);
    });
  });

  describe('getRating', () => {
    it('should get a ticket rating', async () => {
      mockGet.mockResolvedValue({ rating: 4, comment: 'Good', ratedBy: 2 });
      const result = await TicketRatingApi.getRating(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/10/rating');
      expect(result?.rating).toBe(4);
    });
  });

  describe('getRatingStats', () => {
    it('should get rating stats without params', async () => {
      mockGet.mockResolvedValue({ totalRatings: 100, averageRating: 4.2, ratingDistribution: {} });
      const result = await TicketRatingApi.getRatingStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/rating-stats', undefined);
      expect(result.totalRatings).toBe(100);
    });

    it('should get rating stats with params', async () => {
      const params = { assigneeId: 1, startDate: '2024-01-01', endDate: '2024-12-31' };
      mockGet.mockResolvedValue({ totalRatings: 50, averageRating: 4.5, ratingDistribution: {} });
      const result = await TicketRatingApi.getRatingStats(params);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/rating-stats', params);
      expect(result.averageRating).toBe(4.5);
    });
  });
});
