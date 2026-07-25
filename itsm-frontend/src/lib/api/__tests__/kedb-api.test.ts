import { KEDBApi } from '../kedb-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('KEDBApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getKnownErrors', () => {
    it('should get known errors without params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
      await KEDBApi.getKnownErrors();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors', undefined);
    });

    it('should get known errors with params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 });
      await KEDBApi.getKnownErrors({ page: 1, pageSize: 10, status: 'active', keyword: 'dns' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors', { page: 1, pageSize: 10, status: 'active', keyword: 'dns' });
    });
  });

  describe('getKnownError', () => {
    it('should get known error by id', async () => {
      mockGet.mockResolvedValue({ id: 1, title: 'DNS Issue' });
      const result = await KEDBApi.getKnownError(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors/1');
      expect(result.title).toBe('DNS Issue');
    });
  });

  describe('createKnownError', () => {
    it('should create known error', async () => {
      const data = { title: 'New Error', description: 'Desc', rootCause: 'RC' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await KEDBApi.createKnownError(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/known-errors', data);
    });
  });

  describe('updateKnownError', () => {
    it('should update known error', async () => {
      const data = { title: 'Updated', status: 'resolved' };
      mockPut.mockResolvedValue({ id: 1, ...data });
      await KEDBApi.updateKnownError(1, data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/known-errors/1', data);
    });
  });

  describe('deleteKnownError', () => {
    it('should delete known error', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KEDBApi.deleteKnownError(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/known-errors/1');
    });
  });

  describe('getStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ total: 10, active: 5, resolved: 3, deprecated: 2 });
      const result = await KEDBApi.getStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors/stats');
      expect(result.total).toBe(10);
    });
  });

  describe('searchKnownErrors', () => {
    it('should search known errors', async () => {
      mockGet.mockResolvedValue({ knownErrors: [], total: 0 });
      await KEDBApi.searchKnownErrors('dns');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors/search', { q: 'dns' });
    });
  });

  describe('getCategories', () => {
    it('should get categories', async () => {
      mockGet.mockResolvedValue({ categories: ['network', 'application'] });
      const result = await KEDBApi.getCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/known-errors/categories');
      expect(result.categories).toContain('network');
    });
  });

  describe('promoteToKnownError', () => {
    it('should promote to known error', async () => {
      mockPost.mockResolvedValue({ message: 'Promoted' });
      const result = await KEDBApi.promoteToKnownError(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/known-errors/1/promote', {});
      expect(result.message).toBe('Promoted');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(KEDBApi.getKnownError(999)).rejects.toThrow('Not found');
    });
  });
});
