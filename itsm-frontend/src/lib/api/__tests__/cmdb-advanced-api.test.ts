import { CMDBAdvancedApi } from '@/lib/api/cmdb-advanced-api';
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

describe('CMDBAdvancedApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTags', () => {
    it('should get tags', async () => {
      mockGet.mockResolvedValue({ items: [{ id: 1, key: 'env', value: 'prod' }], total: 1, page: 1, size: 10 });
      const result = await CMDBAdvancedApi.getTags({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/tags', { page: 1 });
      expect(result.items).toHaveLength(1);
    });
  });

  describe('createTag', () => {
    it('should create a tag', async () => {
      mockPost.mockResolvedValue({ id: 1, key: 'env', value: 'prod' });
      const result = await CMDBAdvancedApi.createTag({ key: 'env', value: 'prod' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/tags', { key: 'env', value: 'prod' });
    });
  });

  describe('deleteTag', () => {
    it('should delete a tag', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBAdvancedApi.deleteTag(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/tags/1');
    });
  });

  describe('getViews', () => {
    it('should get views', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, size: 10 });
      const result = await CMDBAdvancedApi.getViews();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/views', undefined);
    });
  });

  describe('createView', () => {
    it('should create a view', async () => {
      const data = { name: 'Test View', filters: {} };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await CMDBAdvancedApi.createView(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/views', data);
    });
  });

  describe('deleteView', () => {
    it('should delete a view', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBAdvancedApi.deleteView(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/views/1');
    });
  });

  describe('createImportTask', () => {
    it('should create import task', async () => {
      const data = { fileUrl: '/tmp/file.csv', fileType: 'csv' as const, updateMode: 'skip' as const };
      mockPost.mockResolvedValue({ taskId: 't1', status: 'pending' });
      await CMDBAdvancedApi.createImportTask(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/import', data);
    });
  });

  describe('createExportTask', () => {
    it('should create export task', async () => {
      const data = { exportType: 'csv' as const };
      mockPost.mockResolvedValue({ taskId: 't2', status: 'pending' });
      await CMDBAdvancedApi.createExportTask(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/export', data);
    });
  });

  describe('batchCreateCI', () => {
    it('should batch create CIs', async () => {
      mockPost.mockResolvedValue({ items: [], total: 2, successCount: 2, failedCount: 0 });
      await CMDBAdvancedApi.batchCreateCI([{ name: 'A', ciTypeId: 1, status: 'active' }]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/cis/batch', { items: [{ name: 'A', ciTypeId: 1, status: 'active' }] });
    });
  });

  describe('batchDeleteCI', () => {
    it('should batch delete CIs', async () => {
      mockDelete.mockResolvedValue({ deletedCount: 2, failedIds: [] });
      await CMDBAdvancedApi.batchDeleteCI([1, 2]);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/cis/batch', { ids: [1, 2] });
    });
  });

  describe('getLifecycleHistory', () => {
    it('should get lifecycle history', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await CMDBAdvancedApi.getLifecycleHistory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/cis/1/lifecycle/history');
    });
  });

  describe('updateLifecycleStatus', () => {
    it('should update lifecycle status', async () => {
      mockPut.mockResolvedValue({ ciId: 1, status: 'maintenance' });
      await CMDBAdvancedApi.updateLifecycleStatus(1, 'maintenance', 'Planned update');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/cmdb/cis/1/lifecycle', { status: 'maintenance', reason: 'Planned update' });
    });
  });

  describe('addTagsToCI', () => {
    it('should add tags to CI', async () => {
      mockPost.mockResolvedValue({ ciId: 1, tagIds: [1, 2] });
      await CMDBAdvancedApi.addTagsToCI(1, [1, 2]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/cis/1/tags', { tagIds: [1, 2] });
    });
  });

  describe('removeTagsFromCI', () => {
    it('should remove tags from CI', async () => {
      mockDelete.mockResolvedValue({ ciId: 1, removedTagIds: [1] });
      await CMDBAdvancedApi.removeTagsFromCI(1, [1]);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/cis/1/tags', { tagIds: [1] });
    });
  });

  describe('getRelationships', () => {
    it('should get relationships', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await CMDBAdvancedApi.getRelationships({ sourceCiId: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/relationships', { sourceCiId: 1 });
    });
  });

  describe('deleteRelationship', () => {
    it('should delete a relationship', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBAdvancedApi.deleteRelationship(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/relationships/1');
    });
  });
});
