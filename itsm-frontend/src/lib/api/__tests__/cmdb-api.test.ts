import { CMDBApi } from '@/lib/api/cmdb-api';
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

jest.spyOn(console, 'error').mockImplementation(() => {});

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('CMDBApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getCIs', () => {
    it('should get CI list', async () => {
      mockGet.mockResolvedValue({ items: [{ id: 1, name: 'Server-01' }], total: 1 });
      const result = await CMDBApi.getCIs({ status: 'active' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items', expect.objectContaining({ status: 'active' }));
      expect(result.items).toHaveLength(1);
    });
  });

  describe('getCI', () => {
    it('should get CI by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Server-01' });
      const result = await CMDBApi.getCI(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1');
    });
  });

  describe('createCI', () => {
    it('should create a CI', async () => {
      const data = { name: 'Server-02', ciTypeId: 1, status: 'active' };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await CMDBApi.createCI(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/configuration-items', data);
    });
  });

  describe('updateCI', () => {
    it('should update a CI', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      await CMDBApi.updateCI(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/configuration-items/1', { name: 'Updated' });
    });
  });

  describe('deleteCI', () => {
    it('should delete a CI', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBApi.deleteCI(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/configuration-items/1');
    });
  });

  describe('getCMDBStats', () => {
    it('should get CMDB stats', async () => {
      mockGet.mockResolvedValue({ total: 50 });
      const result = await CMDBApi.getCMDBStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/stats', undefined);
    });
  });

  describe('getCITypes', () => {
    it('should get CI types (array response)', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Server' }]);
      const result = await CMDBApi.getCITypes();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/types');
      expect(result).toHaveLength(1);
    });

    it('should handle object response with items', async () => {
      mockGet.mockResolvedValue({ items: [{ id: 1, name: 'Server' }] });
      const result = await CMDBApi.getCITypes();
      expect(result).toHaveLength(1);
    });
  });

  describe('createCITypes', () => {
    it('should create a CI type', async () => {
      mockPost.mockResolvedValue({ id: 1, name: 'Database' });
      await CMDBApi.createCITypes({ name: 'Database' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/configuration-items/types', { name: 'Database' });
    });
  });

  describe('deleteCITypes', () => {
    it('should delete a CI type', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBApi.deleteCITypes(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/configuration-items/types/1');
    });
  });

  describe('createCIRelationship', () => {
    it('should create CI relationship', async () => {
      const data = { parentId: 1, childId: 2, type: 'depends_on' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await CMDBApi.createCIRelationship(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/configuration-items/relationships', data);
    });
  });

  describe('getCIRelationships', () => {
    it('should get CI relationships', async () => {
      mockGet.mockResolvedValue([{ id: 1, parentId: 1, childId: 2 }]);
      const result = await CMDBApi.getCIRelationships(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/relationships', undefined);
    });
  });

  describe('getCloudServices', () => {
    it('should get cloud services', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'ECS' }]);
      const result = await CMDBApi.getCloudServices('alibaba');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/cloud-services', { provider: 'alibaba' });
    });
  });

  describe('getCloudAccounts', () => {
    it('should get cloud accounts', async () => {
      mockGet.mockResolvedValue([{ id: '1', name: 'Prod Account' }]);
      await CMDBApi.getCloudAccounts();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/cloud-accounts');
    });
  });
});
