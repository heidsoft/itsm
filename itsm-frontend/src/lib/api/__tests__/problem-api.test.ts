import { ProblemApi } from '@/lib/api/problem-api';
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
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockRequest = (httpClient as any).request as jest.Mock;

describe('ProblemApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getProblems', () => {
    it('should get problems list', async () => {
      mockGet.mockResolvedValue({ problems: [{ id: 1, title: 'Memory leak' }], total: 1, page: 1, pageSize: 10 });
      const result = await ProblemApi.getProblems({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems', { page: 1 });
      expect(result.problems).toHaveLength(1);
    });
  });

  describe('getProblem', () => {
    it('should get problem by id', async () => {
      mockGet.mockResolvedValue({ id: 1, title: 'Memory leak' });
      const result = await ProblemApi.getProblem(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/1');
      expect(result.title).toBe('Memory leak');
    });
  });

  describe('createProblem', () => {
    it('should create a problem', async () => {
      const data = { title: 'New Problem', description: 'desc', priority: 'high' };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await ProblemApi.createProblem(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/problems', data);
      expect(result.id).toBe(2);
    });
  });

  describe('updateProblem', () => {
    it('should update a problem', async () => {
      mockPut.mockResolvedValue({ id: 1, title: 'Updated' });
      const result = await ProblemApi.updateProblem(1, { title: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/problems/1', { title: 'Updated' });
      expect(result.title).toBe('Updated');
    });
  });

  describe('deleteProblem', () => {
    it('should delete a problem', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ProblemApi.deleteProblem(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/problems/1');
    });
  });

  describe('getProblemStats', () => {
    it('should get problem stats', async () => {
      mockGet.mockResolvedValue({ total: 10, open: 5 });
      const result = await ProblemApi.getProblemStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/stats', undefined);
      expect(result.total).toBe(10);
    });
  });

  describe('getTrends', () => {
    it('should get trends', async () => {
      const params = { startDate: '2024-01-01', endDate: '2024-12-31' };
      mockGet.mockResolvedValue({ period: '2024', totalProblems: 50 });
      const result = await ProblemApi.getTrends(params);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/trend', params);
      expect(result.totalProblems).toBe(50);
    });
  });

  describe('getHotspots', () => {
    it('should get hotspots', async () => {
      const params = { startDate: '2024-01-01', endDate: '2024-12-31' };
      mockGet.mockResolvedValue({ hotspots: ['network'] });
      const result = await ProblemApi.getHotspots(params);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/hotspots', params);
    });
  });

  describe('getAssociations', () => {
    it('should get associations', async () => {
      mockGet.mockResolvedValue({ tickets: [], incidents: [], changes: [] });
      const result = await ProblemApi.getAssociations(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/1/associations');
    });
  });

  describe('addAssociation', () => {
    it('should add association', async () => {
      mockPost.mockResolvedValue(undefined);
      await ProblemApi.addAssociation(1, { relatedType: 'ticket', relatedIds: [2, 3] });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/problems/1/associations', { relatedType: 'ticket', relatedIds: [2, 3] });
    });
  });

  describe('removeAssociation', () => {
    it('should remove association', async () => {
      mockRequest.mockResolvedValue(undefined);
      await ProblemApi.removeAssociation(1, { relatedType: 'ticket', relatedId: 2 });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'DELETE', url: '/api/v1/problems/1/associations' }));
    });
  });

  describe('getProblemSLA', () => {
    it('should get problem SLA', async () => {
      mockGet.mockResolvedValue({ slaStatus: 'ok', responseBreached: false, resolutionBreached: false, responseTimeUsed: 10, resolutionTimeUsed: 20 });
      const result = await ProblemApi.getProblemSLA(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/problems/1/sla');
      expect(result.slaStatus).toBe('ok');
    });
  });

  describe('stub methods', () => {
    it('investigateProblem should throw', async () => {
      await expect(ProblemApi.investigateProblem(1, {})).rejects.toThrow();
    });
    it('recordRootCause should throw', async () => {
      await expect(ProblemApi.recordRootCause(1, 'cause')).rejects.toThrow();
    });
    it('provideSolution should throw', async () => {
      await expect(ProblemApi.provideSolution(1, 'sol')).rejects.toThrow();
    });
    it('closeProblem should throw', async () => {
      await expect(ProblemApi.closeProblem(1, 'done')).rejects.toThrow();
    });
  });
});
