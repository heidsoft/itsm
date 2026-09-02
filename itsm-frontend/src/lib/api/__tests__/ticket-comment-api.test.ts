import { TicketCommentApi } from '@/lib/api/ticket-comment-api';
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

describe('TicketCommentApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getComments', () => {
    it('should fetch comments for a ticket', async () => {
      mockGet.mockResolvedValue({ items: [{ id: 1, content: 'test' }], total: 1 });
      const result = await TicketCommentApi.getComments(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/10/comments');
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(1);
    });

  });

  describe('createComment', () => {
    it('should create a comment', async () => {
      const payload = { content: 'Hello', isInternal: false };
      mockPost.mockResolvedValue({ id: 1, ...payload });
      const result = await TicketCommentApi.createComment(5, payload);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/5/comments', payload);
      expect(result.content).toBe('Hello');
    });
  });

  describe('updateComment', () => {
    it('should update a comment', async () => {
      const payload = { content: 'Updated' };
      mockPut.mockResolvedValue({ id: 2, content: 'Updated' });
      const result = await TicketCommentApi.updateComment(5, 2, payload);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/5/comments/2', payload);
      expect(result.content).toBe('Updated');
    });
  });

  describe('deleteComment', () => {
    it('should delete a comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketCommentApi.deleteComment(5, 2);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/5/comments/2');
    });
  });
});
