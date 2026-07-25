import { AssetApi } from '@/lib/api/asset-api';
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

describe('AssetApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getAssets', () => {
    it('should get asset list', async () => {
      mockGet.mockResolvedValue({ assets: [{ id: 1, name: 'Server' }], total: 1 });
      const result = await AssetApi.getAssets({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/assets', { page: 1 });
      expect(result.assets).toHaveLength(1);
    });
  });

  describe('getAsset', () => {
    it('should get asset by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Server' });
      const result = await AssetApi.getAsset(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/assets/1');
      expect(result.name).toBe('Server');
    });
  });

  describe('createAsset', () => {
    it('should create an asset', async () => {
      const data = { assetNumber: 'A001', name: 'Laptop' };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await AssetApi.createAsset(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/assets', data);
      expect(result.name).toBe('Laptop');
    });
  });

  describe('updateAsset', () => {
    it('should update an asset', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      const result = await AssetApi.updateAsset(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/assets/1', { name: 'Updated' });
    });
  });

  describe('deleteAsset', () => {
    it('should delete an asset', async () => {
      mockDelete.mockResolvedValue(undefined);
      await AssetApi.deleteAsset(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/assets/1');
    });
  });

  describe('updateAssetStatus', () => {
    it('should update asset status', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'maintenance' });
      await AssetApi.updateAssetStatus(1, 'maintenance', 5);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/assets/1/status', { status: 'maintenance', assignedTo: 5 });
    });
  });

  describe('getAssetStats', () => {
    it('should get asset stats', async () => {
      mockGet.mockResolvedValue({ total: 100, available: 60 });
      const result = await AssetApi.getAssetStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/assets/stats');
      expect(result.total).toBe(100);
    });
  });

  describe('assignAsset', () => {
    it('should assign an asset', async () => {
      mockPut.mockResolvedValue({ id: 1, assignedTo: 10 });
      await AssetApi.assignAsset(1, 10);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/assets/1/assign', { assignedTo: 10 });
    });
  });

  describe('retireAsset', () => {
    it('should retire an asset', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'retired' });
      await AssetApi.retireAsset(1, 'End of life');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/assets/1/retire', { retireReason: 'End of life' });
    });
  });

  describe('getLicenses', () => {
    it('should get license list', async () => {
      mockGet.mockResolvedValue({ licenses: [{ id: 1, name: 'Office' }], total: 1 });
      const result = await AssetApi.getLicenses({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/licenses', { page: 1 });
      expect(result.licenses).toHaveLength(1);
    });
  });

  describe('createLicense', () => {
    it('should create a license', async () => {
      mockPost.mockResolvedValue({ id: 1, name: 'Office 365' });
      const result = await AssetApi.createLicense({ name: 'Office 365' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/licenses', { name: 'Office 365' });
    });
  });

  describe('deleteLicense', () => {
    it('should delete a license', async () => {
      mockDelete.mockResolvedValue(undefined);
      await AssetApi.deleteLicense(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/licenses/1');
    });
  });

  describe('getLicenseStats', () => {
    it('should get license stats', async () => {
      mockGet.mockResolvedValue({ total: 50, active: 40 });
      const result = await AssetApi.getLicenseStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/licenses/stats');
      expect(result.total).toBe(50);
    });
  });

  describe('assignLicenseUsers', () => {
    it('should assign license to users', async () => {
      mockPut.mockResolvedValue({ id: 1, users: [1, 2] });
      await AssetApi.assignLicenseUsers(1, [1, 2]);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/licenses/1/assign', { userIds: [1, 2] });
    });
  });
});
