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

  describe('getCMDBTypes', () => {
    it('should delegate to getCITypes', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Server' }]);
      const result = await CMDBApi.getCMDBTypes();
      expect(result).toHaveLength(1);
    });
  });

  describe('updateCITypes', () => {
    it('should update CI type', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      await CMDBApi.updateCITypes(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/configuration-items/types/1', { name: 'Updated' });
    });
  });

  describe('getCITopology', () => {
    it('should get topology', async () => {
      mockGet.mockResolvedValue({ nodes: [], edges: [] });
      await CMDBApi.getCITopology(1, 5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/topology', { depth: 5 });
    });
  });

  describe('getCIImpactAnalysis', () => {
    it('should get impact analysis', async () => {
      mockGet.mockResolvedValue({ impactedItems: [] });
      await CMDBApi.getCIImpactAnalysis(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/impact-analysis');
    });
  });

  describe('analyzeImpact', () => {
    it('should analyze impact with maxDepth', async () => {
      mockGet.mockResolvedValue({ impactedItems: [] });
      await CMDBApi.analyzeImpact({ ciId: '1', maxDepth: 5 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/impact-analysis', { maxDepth: 5 });
    });
  });

  describe('getCIChangeHistory', () => {
    it('should get change history', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await CMDBApi.getCIChangeHistory(1, { page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/change-history', { page: 1, pageSize: 10 });
    });
  });

  describe('createRelationship', () => {
    it('should create relationship mapping source/target to parent/child', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await CMDBApi.createRelationship({ sourceCiId: 10, targetCiId: 20, type: 'depends_on', description: 'test' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/configuration-items/relationships', { parentId: 10, childId: 20, type: 'depends_on', description: 'test' });
    });
  });

  describe('deleteRelationship', () => {
    it('should delete relationship', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBApi.deleteRelationship('5');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/configuration-items/relationships/5');
    });
  });

  describe('getReconciliationResults', () => {
    it('should get reconciliation results', async () => {
      mockGet.mockResolvedValue({ results: [] });
      await CMDBApi.getReconciliationResults({ status: 'pending' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/reconciliation', { status: 'pending' });
    });
  });

  describe('createCloudService', () => {
    it('should create cloud service', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await CMDBApi.createCloudService({ name: 'EC2', provider: 'aws' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/cloud-services', { name: 'EC2', provider: 'aws' });
    });
  });

  describe('updateCloudService', () => {
    it('should update cloud service', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await CMDBApi.updateCloudService(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/cmdb/cloud-services/1', { name: 'Updated' });
    });
  });

  describe('deleteCloudService', () => {
    it('should delete cloud service', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBApi.deleteCloudService(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/cloud-services/1');
    });
  });

  describe('createCloudAccount', () => {
    it('should create cloud account', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await CMDBApi.createCloudAccount({ name: 'Prod', provider: 'aws' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/cloud-accounts', { name: 'Prod', provider: 'aws' });
    });
  });

  describe('deleteCloudAccount', () => {
    it('should delete cloud account', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CMDBApi.deleteCloudAccount('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/cmdb/cloud-accounts/1');
    });
  });

  describe('updateCloudAccount', () => {
    it('should update cloud account', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await CMDBApi.updateCloudAccount('1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/cmdb/cloud-accounts/1', { name: 'Updated' });
    });
  });

  describe('getCloudResources', () => {
    it('should get cloud resources', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getCloudResources({ accountId: '1' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/cloud-resources', { accountId: '1' });
    });
  });

  describe('getDiscoveryRules', () => {
    it('should get discovery rules', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getDiscoveryRules();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/discovery/sources');
    });
  });

  describe('getDiscoverySources', () => {
    it('should get discovery sources', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getDiscoverySources();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/discovery/sources');
    });
  });

  describe('getDiscoveryHistory', () => {
    it('should get discovery history with ruleId', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getDiscoveryHistory('rule1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/discovery/results', { jobId: 'rule1' });
    });

    it('should get all discovery history without ruleId', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getDiscoveryHistory();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/discovery/results', undefined);
    });
  });

  describe('runDiscoveryRule', () => {
    it('should run discovery rule', async () => {
      mockPost.mockResolvedValue(undefined);
      await CMDBApi.runDiscoveryRule('rule1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/cmdb/discovery/jobs', { sourceId: 'rule1' });
    });
  });

  describe('searchCIs', () => {
    it('should search CIs', async () => {
      mockGet.mockResolvedValue({ items: [{ id: 1 }], total: 1 });
      const result = await CMDBApi.searchCIs({ keyword: 'server' });
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(1);
    });
  });

  describe('batchCreateCIs', () => {
    it('should batch create CIs', async () => {
      mockPost.mockResolvedValue({ id: 1, name: 'S1' });
      const result = await CMDBApi.batchCreateCIs([{ name: 'S1', ciTypeId: 1, status: 'active' }, { name: 'S2', ciTypeId: 1, status: 'active' }]);
      expect(result).toHaveLength(2);
    });

    it('should continue on individual failure', async () => {
      mockPost.mockResolvedValueOnce({ id: 1, name: 'S1' }).mockRejectedValueOnce(new Error('fail'));
      const result = await CMDBApi.batchCreateCIs([{ name: 'S1', ciTypeId: 1, status: 'active' }, { name: 'S2', ciTypeId: 1, status: 'active' }]);
      expect(result).toHaveLength(1);
    });
  });

  describe('getCloudServices without provider', () => {
    it('should get all cloud services', async () => {
      mockGet.mockResolvedValue([]);
      await CMDBApi.getCloudServices();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/cmdb/cloud-services', undefined);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors on getCI', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(CMDBApi.getCI(999)).rejects.toThrow('Not found');
    });
  });
});
