import { TicketViewApi } from '@/lib/api/ticket-view-api';
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

describe('TicketViewApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('listViews', () => {
    it('should list ticket views', async () => {
      mockGet.mockResolvedValue({ views: [{ id: 1, name: 'My View' }], total: 1 });
      const result = await TicketViewApi.listViews();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/views');
      expect(result.views).toHaveLength(1);
    });
  });

  describe('getView', () => {
    it('should get a view by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'My View' });
      const result = await TicketViewApi.getView(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/views/1');
      expect(result.name).toBe('My View');
    });
  });

  describe('createView', () => {
    it('should create a view', async () => {
      const data = { name: 'New View', filters: {}, columns: ['id'], sortConfig: { field: 'id', order: 'asc' as const }, isShared: false };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await TicketViewApi.createView(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/views', data);
      expect(result.name).toBe('New View');
    });
  });

  describe('updateView', () => {
    it('should update a view', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated View' });
      const result = await TicketViewApi.updateView(1, { name: 'Updated View' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/views/1', { name: 'Updated View' });
      expect(result.name).toBe('Updated View');
    });
  });

  describe('deleteView', () => {
    it('should delete a view', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketViewApi.deleteView(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/views/1');
    });
  });

  describe('shareView', () => {
    it('should share a view with teams', async () => {
      const data = { teamIds: [1, 2] };
      mockPost.mockResolvedValue(undefined);
      await TicketViewApi.shareView(1, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/views/1/share', data);
    });
  });
});
