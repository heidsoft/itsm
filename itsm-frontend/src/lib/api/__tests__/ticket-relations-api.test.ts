import { TicketRelationsApi } from '../ticket-relations-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
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
const mockRequest = httpClient.request as jest.Mock;

describe('TicketRelationsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('createRelation', () => {
    it('should create relation', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TicketRelationsApi.createRelation({ sourceTicketId: 1, targetTicketId: 2, relationType: 'blocks' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations', expect.any(Object));
    });
  });

  describe('batchCreateRelations', () => {
    it('should batch create', async () => {
      mockPost.mockResolvedValue({ created: 2, failed: 0 });
      await TicketRelationsApi.batchCreateRelations({ relations: [] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/batch', expect.any(Object));
    });
  });

  describe('getTicketRelations', () => {
    it('should get relations', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.getTicketRelations(1, { relationType: 'blocks' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations', { relationType: 'blocks' });
    });
  });

  describe('getRelation', () => {
    it('should get relation by id', async () => {
      mockGet.mockResolvedValue({ id: 'r1' });
      await TicketRelationsApi.getRelation('r1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/relations/r1');
    });
  });

  describe('updateRelation', () => {
    it('should update relation', async () => {
      mockPut.mockResolvedValue({ id: 'r1' });
      await TicketRelationsApi.updateRelation('r1', { description: 'updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/relations/r1', { description: 'updated' });
    });
  });

  describe('deleteRelation', () => {
    it('should delete relation', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketRelationsApi.deleteRelation('r1', 'not needed');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/relations/r1', { reason: 'not needed' });
    });
  });

  describe('batchDeleteRelations', () => {
    it('should batch delete', async () => {
      mockRequest.mockResolvedValue({ deleted: 2 });
      await TicketRelationsApi.batchDeleteRelations(['r1', 'r2'], 'cleanup');
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/tickets/relations/batch', data: { relationIds: ['r1', 'r2'], reason: 'cleanup' } });
    });
  });

  describe('setParent', () => {
    it('should set parent', async () => {
      mockPost.mockResolvedValue({ id: 'r1' });
      await TicketRelationsApi.setParent(2, 1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/2/parent', { parentTicketId: 1 });
    });
  });

  describe('removeParent', () => {
    it('should remove parent', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketRelationsApi.removeParent(2);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/2/parent');
    });
  });

  describe('getChildren', () => {
    it('should get children', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.getChildren(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/children');
    });
  });

  describe('getHierarchy', () => {
    it('should get hierarchy', async () => {
      mockGet.mockResolvedValue({ root: 1 });
      await TicketRelationsApi.getHierarchy(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/hierarchy');
    });
  });

  describe('addDependency', () => {
    it('should add dependency', async () => {
      mockPost.mockResolvedValue({ id: 'd1' });
      await TicketRelationsApi.addDependency({ ticketId: 1, dependsOnTicketId: 2, dependencyType: 'hard' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/dependencies', { dependsOnTicketId: 2, dependencyType: 'hard' });
    });
  });

  describe('removeDependency', () => {
    it('should remove dependency', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketRelationsApi.removeDependency(1, 'd1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/dependencies/d1');
    });
  });

  describe('getDependencies', () => {
    it('should get dependencies', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.getDependencies(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/dependencies');
    });
  });

  describe('getDependencyGraph', () => {
    it('should get graph', async () => {
      mockGet.mockResolvedValue({ nodes: [], edges: [] });
      await TicketRelationsApi.getDependencyGraph(1, 3);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/dependency-graph', { maxDepth: 3 });
    });
  });

  describe('canStartWork', () => {
    it('should check can start', async () => {
      mockGet.mockResolvedValue({ canStart: true, blockingTickets: [] });
      await TicketRelationsApi.canStartWork(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/can-start');
    });
  });

  describe('validateRelation', () => {
    it('should validate', async () => {
      mockPost.mockResolvedValue({ isValid: true });
      await TicketRelationsApi.validateRelation({ sourceTicketId: 1, targetTicketId: 2 } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/validate', expect.any(Object));
    });
  });

  describe('detectCircularDependency', () => {
    it('should detect circular', async () => {
      mockPost.mockResolvedValue({ hasCircular: false });
      await TicketRelationsApi.detectCircularDependency({ sourceTicketId: 1, targetTicketId: 2 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/detect-circular', { sourceTicketId: 1, targetTicketId: 2 });
    });
  });

  describe('getConflicts', () => {
    it('should get conflicts', async () => {
      mockPost.mockResolvedValue([]);
      await TicketRelationsApi.getConflicts({ sourceTicketId: 1 } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/conflicts', expect.any(Object));
    });
  });

  describe('searchRelations', () => {
    it('should search relations', async () => {
      mockGet.mockResolvedValue({ relations: [], total: 0 });
      await TicketRelationsApi.searchRelations({ ticketId: 1, search: 'test' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/relations/search', { ticketId: 1, search: 'test' });
    });
  });

  describe('findRelatedTickets', () => {
    it('should find related tickets', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.findRelatedTickets(1, { maxResults: 5 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/related', { maxResults: 5 });
    });
  });

  describe('findDuplicates', () => {
    it('should find duplicates', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.findDuplicates(1, 0.8);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/duplicates', { threshold: 0.8 });
    });
  });

  describe('getRelationStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalRelations: 5 });
      await TicketRelationsApi.getRelationStats(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations/stats');
    });
  });

  describe('analyzeImpact', () => {
    it('should analyze impact', async () => {
      mockPost.mockResolvedValue({ affectedTickets: [] });
      await TicketRelationsApi.analyzeImpact({ ticketId: 1, action: 'close' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/impact-analysis', { action: 'close', newStatus: undefined });
    });
  });

  describe('getCriticalPath', () => {
    it('should get critical path', async () => {
      mockGet.mockResolvedValue({ path: [1, 2, 3] });
      await TicketRelationsApi.getCriticalPath(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/critical-path');
    });
  });

  describe('getRelationGraph', () => {
    it('should get graph data', async () => {
      mockGet.mockResolvedValue({ nodes: [], edges: [] });
      await TicketRelationsApi.getRelationGraph(1, { maxDepth: 2 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/graph', { maxDepth: 2 });
    });
  });

  describe('getRelationSuggestions', () => {
    it('should get suggestions', async () => {
      mockGet.mockResolvedValue({ suggestions: [] });
      await TicketRelationsApi.getRelationSuggestions(1, 5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relation-suggestions', { limit: 5 });
    });
  });

  describe('getAIRecommendations', () => {
    it('should get AI recommendations', async () => {
      mockGet.mockResolvedValue([]);
      await TicketRelationsApi.getAIRecommendations(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/ai-recommendations');
    });
  });

  describe('getRelationHistory', () => {
    it('should get history', async () => {
      mockGet.mockResolvedValue({ history: [], total: 0 });
      await TicketRelationsApi.getRelationHistory(1, { page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations/history', { page: 1 });
    });
  });

  describe('getRelationPermissions', () => {
    it('should get permissions', async () => {
      mockGet.mockResolvedValue({ canCreate: true });
      await TicketRelationsApi.getRelationPermissions(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations/permissions');
    });
  });

  describe('canCreateRelation', () => {
    it('should check can create', async () => {
      mockPost.mockResolvedValue({ canCreate: true });
      await TicketRelationsApi.canCreateRelation({ sourceTicketId: 1, targetTicketId: 2, relationType: 'blocks' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/can-create', { sourceTicketId: 1, targetTicketId: 2, relationType: 'blocks' });
    });
  });

  describe('copyRelations', () => {
    it('should copy relations', async () => {
      mockPost.mockResolvedValue({ created: 3 });
      await TicketRelationsApi.copyRelations({ sourceTicketId: 1, targetTicketId: 2 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/copy', { sourceTicketId: 1, targetTicketId: 2 });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(TicketRelationsApi.getRelation('999')).rejects.toThrow('Not found');
    });
  });
});
