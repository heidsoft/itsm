import { TicketRelationsApi } from '@/lib/api/ticket-relations-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
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
const mockRequest = (httpClient as any).request as jest.Mock;

describe('TicketRelationsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('createRelation', () => {
    it('should create a relation', async () => {
      const req = { sourceTicketId: 1, targetTicketId: 2, relationType: 'related' };
      mockPost.mockResolvedValue({ id: '1', ...req });
      const result = await TicketRelationsApi.createRelation(req as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations', req);
      expect(result.id).toBe('1');
    });
  });

  describe('batchCreateRelations', () => {
    it('should batch create relations', async () => {
      mockPost.mockResolvedValue({ success: 2, failed: 0 });
      const result = await TicketRelationsApi.batchCreateRelations({ relations: [] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/batch', expect.any(Object));
    });
  });

  describe('getTicketRelations', () => {
    it('should get ticket relations', async () => {
      mockGet.mockResolvedValue([{ id: '1' }]);
      const result = await TicketRelationsApi.getTicketRelations(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations', undefined);
      expect(result).toHaveLength(1);
    });
  });

  describe('getRelation', () => {
    it('should get a single relation', async () => {
      mockGet.mockResolvedValue({ id: 'r1' });
      const result = await TicketRelationsApi.getRelation('r1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/relations/r1');
    });
  });

  describe('updateRelation', () => {
    it('should update a relation', async () => {
      mockPut.mockResolvedValue({ id: 'r1' });
      await TicketRelationsApi.updateRelation('r1', {} as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/relations/r1', expect.any(Object));
    });
  });

  describe('deleteRelation', () => {
    it('should delete a relation', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketRelationsApi.deleteRelation('r1', 'no longer needed');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/relations/r1', { reason: 'no longer needed' });
    });
  });

  describe('setParent', () => {
    it('should set parent ticket', async () => {
      mockPost.mockResolvedValue({ id: '1' });
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

  describe('validateRelation', () => {
    it('should validate a relation', async () => {
      mockPost.mockResolvedValue({ isValid: true });
      await TicketRelationsApi.validateRelation({} as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/validate', expect.any(Object));
    });
  });

  describe('searchRelations', () => {
    it('should search relations', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await TicketRelationsApi.searchRelations({ ticketId: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/relations/search', { ticketId: 1 });
    });
  });

  describe('getRelationStats', () => {
    it('should get relation stats', async () => {
      mockGet.mockResolvedValue({ total: 5 });
      await TicketRelationsApi.getRelationStats(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/relations/stats');
    });
  });

  describe('canCreateRelation', () => {
    it('should check if can create relation', async () => {
      mockPost.mockResolvedValue({ canCreate: true });
      const result = await TicketRelationsApi.canCreateRelation({ sourceTicketId: 1, targetTicketId: 2, relationType: 'related' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/relations/can-create', expect.any(Object));
      expect(result.canCreate).toBe(true);
    });
  });
});
