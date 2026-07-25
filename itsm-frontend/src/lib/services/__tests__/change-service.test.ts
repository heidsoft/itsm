/**
 * ChangeService unit tests
 */
import { changeService } from '../change-service';
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

describe('ChangeService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getChanges', () => {
    it('should call GET /api/v1/changes with params', async () => {
      const mockData = { changes: [], total: 0, page: 1, pageSize: 20, totalPages: 0 };
      mockGet.mockResolvedValueOnce(mockData);

      const result = await changeService.getChanges({ page: 1, pageSize: 10, status: 'pending' });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes', { page: 1, pageSize: 10, status: 'pending' });
      expect(result).toEqual(mockData);
    });

    it('should call with empty params when none provided', async () => {
      const mockData = { changes: [], total: 0, page: 1, pageSize: 20, totalPages: 0 };
      mockGet.mockResolvedValueOnce(mockData);

      await changeService.getChanges({});

      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes', {});
    });
  });

  describe('getChange', () => {
    it('should call GET /api/v1/changes/:id', async () => {
      const mockChange = { id: 1, title: 'DB Upgrade' };
      mockGet.mockResolvedValueOnce(mockChange);

      const result = await changeService.getChange(1);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/1');
      expect(result).toEqual(mockChange);
    });
  });

  describe('createChange', () => {
    it('should call POST /api/v1/changes with data', async () => {
      const createData = {
        title: 'New Change',
        description: 'Desc',
        justification: 'Just',
        type: 'normal' as const,
        priority: 'high' as const,
        impactScope: 'medium' as const,
        riskLevel: 'low' as const,
        implementationPlan: 'plan',
        rollbackPlan: 'rollback',
      };
      const mockResult = { id: 2, ...createData };
      mockPost.mockResolvedValueOnce(mockResult);

      const result = await changeService.createChange(createData);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/changes', createData);
      expect(result).toEqual(mockResult);
    });
  });

  describe('updateChange', () => {
    it('should call PUT /api/v1/changes/:id with data', async () => {
      const updateData = { title: 'Updated' };
      mockPut.mockResolvedValueOnce({ id: 1, title: 'Updated' });

      const result = await changeService.updateChange(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/changes/1', updateData);
      expect(result.title).toBe('Updated');
    });
  });

  describe('deleteChange', () => {
    it('should call DELETE /api/v1/changes/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);

      await changeService.deleteChange(5);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/changes/5');
    });
  });

  describe('getChangeStats', () => {
    it('should call GET /api/v1/changes/stats', async () => {
      const mockStats = { total: 10, draft: 2, pending: 3, approved: 2, implementing: 1, completed: 1, cancelled: 1 };
      mockGet.mockResolvedValueOnce(mockStats);

      const result = await changeService.getChangeStats();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/stats');
      expect(result).toEqual(mockStats);
    });
  });

  describe('healthCheck', () => {
    it('should return true on success', async () => {
      mockGet.mockResolvedValueOnce({ status: 'ok' });

      const result = await changeService.healthCheck();

      expect(result).toBe(true);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/health');
    });

    it('should return false on failure', async () => {
      mockGet.mockRejectedValueOnce(new Error('Network error'));

      const result = await changeService.healthCheck();

      expect(result).toBe(false);
    });
  });
});
