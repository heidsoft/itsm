import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
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

describe('ServiceCatalogApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getServices', () => {
    it('should get services list', async () => {
      mockGet.mockResolvedValue({ catalogs: [{ id: 1, name: 'Email', status: 'enabled', category: 'it_service' }], total: 1 });
      const result = await ServiceCatalogApi.getServices({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs', expect.objectContaining({ page: 1, size: 10 }));
      expect(result.services).toHaveLength(1);
    });
  });

  describe('getService', () => {
    it('should get a single service', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Email', status: 'enabled' });
      const result = await ServiceCatalogApi.getService('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/1');
      expect(result.name).toBe('Email');
    });
  });

  describe('createService', () => {
    it('should create a service', async () => {
      mockPost.mockResolvedValue({ id: 2, name: 'VPN', status: 'enabled' });
      const result = await ServiceCatalogApi.createService({ name: 'VPN', category: 'it_service' as any } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-catalogs', expect.objectContaining({ name: 'VPN' }));
    });
  });

  describe('updateService', () => {
    it('should update a service', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated', status: 'enabled' });
      const result = await ServiceCatalogApi.updateService('1', { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', expect.objectContaining({ name: 'Updated' }));
    });
  });

  describe('deleteService', () => {
    it('should delete a service', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ServiceCatalogApi.deleteService('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/service-catalogs/1');
    });
  });

  describe('publishService', () => {
    it('should publish a service', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'enabled' });
      await ServiceCatalogApi.publishService('1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', { status: 'enabled' });
    });
  });

  describe('retireService', () => {
    it('should retire a service', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'disabled' });
      await ServiceCatalogApi.retireService('1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', { status: 'disabled' });
    });
  });

  describe('getServiceRequests', () => {
    it('should get service requests', async () => {
      mockGet.mockResolvedValue({ requests: [{ id: 1 }], total: 1 });
      const result = await ServiceCatalogApi.getServiceRequests({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalled();
      expect(result.total).toBe(1);
    });
  });

  describe('cancelServiceRequest', () => {
    it('should cancel a service request', async () => {
      mockPut.mockResolvedValue(undefined);
      await ServiceCatalogApi.cancelServiceRequest(1, 'No longer needed');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-requests/1/status', { status: 'cancelled', comment: 'No longer needed' });
    });
  });

  describe('approveServiceRequest', () => {
    it('should approve a service request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.approveServiceRequest(1, 'Looks good');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'approve', comment: 'Looks good' });
    });
  });

  describe('rejectServiceRequest', () => {
    it('should reject a service request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.rejectServiceRequest(1, 'Budget issue');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'reject', comment: 'Budget issue' });
    });
  });

  describe('getCatalogStats', () => {
    it('should get catalog stats', async () => {
      mockGet.mockResolvedValue({ totalServices: 20, publishedServices: 15, categories: {} });
      const result = await ServiceCatalogApi.getCatalogStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/stats');
      expect(result.totalServices).toBe(20);
    });
  });

  describe('getFavorites', () => {
    it('should return empty array (not implemented)', async () => {
      const result = await ServiceCatalogApi.getFavorites();
      expect(result).toEqual([]);
    });
  });

  describe('getPortalConfig', () => {
    it('should return default config', async () => {
      const result = await ServiceCatalogApi.getPortalConfig();
      expect(result.name).toBe('默认门户');
    });
  });
});
