/**
 * TicketTagService unit tests
 */
import { ticketTagService } from '../ticket-tag-service';
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

describe('TicketTagService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listTags', () => {
    it('should call GET /api/v1/ticket-tags with params', async () => {
      mockGet.mockResolvedValueOnce({ tags: [{ id: 1, name: 'urgent' }], total: 1 });
      const result = await ticketTagService.listTags({ page: 1, isActive: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-tags', { page: 1, isActive: true });
      expect(result.total).toBe(1);
    });

    it('should use empty params by default', async () => {
      mockGet.mockResolvedValueOnce({ tags: [], total: 0 });
      await ticketTagService.listTags();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-tags', {});
    });
  });

  describe('getTag', () => {
    it('should call GET /api/v1/ticket-tags/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 3, name: 'bug', color: '#ff0000' });
      const result = await ticketTagService.getTag(3);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-tags/3');
      expect(result.color).toBe('#ff0000');
    });
  });

  describe('createTag', () => {
    it('should call POST /api/v1/ticket-tags', async () => {
      const data = { name: 'feature', color: '#00ff00' };
      mockPost.mockResolvedValueOnce({ id: 5, ...data });
      const result = await ticketTagService.createTag(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ticket-tags', data);
      expect(result.id).toBe(5);
    });
  });

  describe('updateTag', () => {
    it('should call PUT /api/v1/ticket-tags/:id', async () => {
      mockPut.mockResolvedValueOnce({ id: 3, name: 'updated', color: '#0000ff' });
      const result = await ticketTagService.updateTag(3, { name: 'updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/ticket-tags/3', { name: 'updated' });
      expect(result.name).toBe('updated');
    });
  });

  describe('deleteTag', () => {
    it('should call DELETE /api/v1/ticket-tags/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await ticketTagService.deleteTag(7);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/ticket-tags/7');
    });
  });
});
