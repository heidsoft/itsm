import { CIRelationshipAPI } from '@/lib/api/cmdb-relationship';
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

describe('CIRelationshipAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getRelationshipTypes', () => {
    it('should get relationship types (array response)', async () => {
      mockGet.mockResolvedValue([{ type: 'depends_on', name: 'Depends On' }]);
      const result = await CIRelationshipAPI.getRelationshipTypes();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/relationship-types');
      expect(result).toHaveLength(1);
    });

    it('should handle object response with types', async () => {
      mockGet.mockResolvedValue({ types: [{ type: 'hosts', name: 'Hosts' }] });
      const result = await CIRelationshipAPI.getRelationshipTypes();
      expect(result).toHaveLength(1);
    });
  });

  describe('createRelationship', () => {
    it('should create a relationship', async () => {
      const data = { sourceCiId: 1, targetCiId: 2, relationshipType: 'depends_on' as const };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await CIRelationshipAPI.createRelationship(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/configuration-items/relationships', data);
    });
  });

  describe('updateRelationship', () => {
    it('should update a relationship', async () => {
      mockPut.mockResolvedValue({ id: 1, relationshipType: 'hosts' });
      await CIRelationshipAPI.updateRelationship(1, { relationshipType: 'hosts' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/configuration-items/relationships/1', { relationshipType: 'hosts' });
    });
  });

  describe('deleteRelationship', () => {
    it('should delete a relationship', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CIRelationshipAPI.deleteRelationship(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/configuration-items/relationships/1');
    });
  });

  describe('getCIRelationships', () => {
    it('should get CI relationships and split by direction', async () => {
      mockGet.mockResolvedValue([
        { id: 1, sourceCiId: 10, targetCiId: 20 },
        { id: 2, sourceCiId: 30, targetCiId: 10 },
      ]);
      const result = await CIRelationshipAPI.getCIRelationships(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/10/relationships', undefined);
      expect(result.outgoingRelations).toHaveLength(1);
      expect(result.incomingRelations).toHaveLength(1);
      expect(result.totalOutgoing).toBe(1);
      expect(result.totalIncoming).toBe(1);
    });
  });

  describe('getTopologyGraph', () => {
    it('should get topology graph', async () => {
      mockGet.mockResolvedValue({ nodes: [], edges: [], rootCiId: 1, depth: 3, totalNodes: 0, totalEdges: 0 });
      const result = await CIRelationshipAPI.getTopologyGraph(1, 3);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/topology?depth=3');
    });

    it('should get topology graph without depth', async () => {
      mockGet.mockResolvedValue({ nodes: [], edges: [] });
      await CIRelationshipAPI.getTopologyGraph(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/topology');
    });
  });

  describe('analyzeImpact', () => {
    it('should analyze impact', async () => {
      mockGet.mockResolvedValue({ sourceCiId: 1, totalImpacted: 5 });
      const result = await CIRelationshipAPI.analyzeImpact(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/configuration-items/1/impact-analysis');
      expect(result.totalImpacted).toBe(5);
    });
  });

  describe('batchCreateRelationships', () => {
    it('should batch create relationships', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      const rels = [
        { sourceCiId: 1, targetCiId: 2, relationshipType: 'depends_on' as const },
        { sourceCiId: 1, targetCiId: 3, relationshipType: 'hosts' as const },
      ];
      const result = await CIRelationshipAPI.batchCreateRelationships(rels);
      expect(mockPost).toHaveBeenCalledTimes(2);
      expect(result.createdCount).toBe(2);
      expect(result.failedCount).toBe(0);
    });
  });
});
