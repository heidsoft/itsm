import { ReleaseApi } from '@/lib/api/release-api';
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

describe('ReleaseApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getReleases', () => {
    it('should get release list', async () => {
      mockGet.mockResolvedValue({ releases: [{ id: 1, title: 'v1.0' }], total: 1 });
      const result = await ReleaseApi.getReleases({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/releases', { page: 1, pageSize: 10 });
      expect(result.releases).toHaveLength(1);
    });
  });

  describe('getRelease', () => {
    it('should get release by id', async () => {
      mockGet.mockResolvedValue({ id: 1, title: 'v1.0' });
      const result = await ReleaseApi.getRelease(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/releases/1');
      expect(result.id).toBe(1);
    });
  });

  describe('createRelease', () => {
    it('should create a release', async () => {
      const data = { releaseNumber: 'R-001', title: 'v1.0' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await ReleaseApi.createRelease(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/releases', data);
      expect(result.title).toBe('v1.0');
    });
  });

  describe('updateRelease', () => {
    it('should update a release', async () => {
      mockPut.mockResolvedValue({ id: 1, title: 'v1.1' });
      const result = await ReleaseApi.updateRelease(1, { title: 'v1.1' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/releases/1', { title: 'v1.1' });
      expect(result.title).toBe('v1.1');
    });
  });

  describe('deleteRelease', () => {
    it('should delete a release', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ReleaseApi.deleteRelease(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/releases/1');
    });
  });

  describe('updateReleaseStatus', () => {
    it('should update release status', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'completed' });
      const result = await ReleaseApi.updateReleaseStatus(1, 'completed');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/releases/1/status', { status: 'completed' });
    });
  });

  describe('approveRelease', () => {
    it('should approve a release', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'scheduled' });
      await ReleaseApi.approveRelease(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/releases/1/approve');
    });
  });

  describe('rejectRelease', () => {
    it('should reject a release', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'cancelled' });
      await ReleaseApi.rejectRelease(1, 'Not ready');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/releases/1/reject', { reason: 'Not ready' });
    });
  });

  describe('rollbackRelease', () => {
    it('should rollback a release', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'rolled_back' });
      await ReleaseApi.rollbackRelease(1, 'Bugs found');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/releases/1/rollback', { reason: 'Bugs found' });
    });
  });

  describe('getReleaseStats', () => {
    it('should get release stats', async () => {
      mockGet.mockResolvedValue({ total: 10, completed: 5 });
      const result = await ReleaseApi.getReleaseStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/releases/stats');
      expect(result.total).toBe(10);
    });
  });
});
