import { CollaborationApi } from '../collaboration-api';
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

describe('CollaborationApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getComments', () => {
    it('should get comments', async () => {
      mockGet.mockResolvedValue({ comments: [], total: 0 });
      await CollaborationApi.getComments({ ticketId: 1, page: 1, pageSize: 10 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/comments', expect.any(Object));
    });
  });

  describe('getComment', () => {
    it('should get single comment', async () => {
      mockGet.mockResolvedValue({ id: '1', content: 'hi' });
      await CollaborationApi.getComment('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/comments/1');
    });
  });

  describe('createComment', () => {
    it('should create comment', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await CollaborationApi.createComment({ ticketId: 1, content: 'test' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/comments', expect.objectContaining({ content: 'test' }));
    });
  });

  describe('updateComment', () => {
    it('should update comment', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await CollaborationApi.updateComment('1', { content: 'updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/comments/1', { content: 'updated' });
    });
  });

  describe('deleteComment', () => {
    it('should delete comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.deleteComment('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/comments/1');
    });
  });

  describe('likeComment', () => {
    it('should like comment', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.likeComment('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/comments/1/like');
    });
  });

  describe('unlikeComment', () => {
    it('should unlike comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.unlikeComment('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/comments/1/like');
    });
  });

  describe('pinComment', () => {
    it('should pin comment', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.pinComment('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/comments/1/pin');
    });
  });

  describe('unpinComment', () => {
    it('should unpin comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.unpinComment('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/comments/1/pin');
    });
  });

  describe('getCommentReplies', () => {
    it('should get replies', async () => {
      mockGet.mockResolvedValue([]);
      await CollaborationApi.getCommentReplies('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/comments/1/replies');
    });
  });

  describe('getCommentStats', () => {
    it('should get comment stats', async () => {
      mockGet.mockResolvedValue({ totalComments: 10 });
      await CollaborationApi.getCommentStats(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/comments/stats');
    });
  });

  describe('searchMentionSuggestions', () => {
    it('should search mentions', async () => {
      mockGet.mockResolvedValue([]);
      await CollaborationApi.searchMentionSuggestions({ query: 'john' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/mentions/suggestions', { query: 'john' });
    });
  });

  describe('getMyMentions', () => {
    it('should get my mentions', async () => {
      mockGet.mockResolvedValue({ mentions: [], total: 0 });
      await CollaborationApi.getMyMentions({ isRead: false });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/mentions/me', { isRead: false });
    });
  });

  describe('markMentionAsRead', () => {
    it('should mark mention as read', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.markMentionAsRead('m1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/mentions/m1/read');
    });
  });

  describe('getNotifications', () => {
    it('should get notifications', async () => {
      mockGet.mockResolvedValue({ notifications: [], total: 0 });
      await CollaborationApi.getNotifications({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notifications', { page: 1 });
    });
  });

  describe('getUnreadCount', () => {
    it('should get unread count', async () => {
      mockGet.mockResolvedValue({ count: 5 });
      const result = await CollaborationApi.getUnreadCount();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notifications/unread-count');
      expect(result.count).toBe(5);
    });
  });

  describe('markNotificationAsRead', () => {
    it('should mark as read', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.markNotificationAsRead('n1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notifications/n1/read');
    });
  });

  describe('markAllNotificationsAsRead', () => {
    it('should mark all as read', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.markAllNotificationsAsRead();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notifications/mark-all-read');
    });
  });

  describe('deleteNotification', () => {
    it('should delete notification', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.deleteNotification('n1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/notifications/n1');
    });
  });

  describe('clearAllNotifications', () => {
    it('should clear all', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.clearAllNotifications();
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/notifications/clear-all');
    });
  });

  describe('getNotificationSettings', () => {
    it('should get settings', async () => {
      mockGet.mockResolvedValue({ emailEnabled: true });
      await CollaborationApi.getNotificationSettings();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notifications/settings');
    });
  });

  describe('updateNotificationSettings', () => {
    it('should update settings', async () => {
      mockPut.mockResolvedValue({ emailEnabled: false });
      await CollaborationApi.updateNotificationSettings({ emailEnabled: false } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notifications/settings', { emailEnabled: false });
    });
  });

  describe('getActivities', () => {
    it('should get activities', async () => {
      mockGet.mockResolvedValue({ activities: [] });
      await CollaborationApi.getActivities({ ticketId: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/activities', expect.any(Object));
    });
  });

  describe('getActivity', () => {
    it('should get single activity', async () => {
      mockGet.mockResolvedValue({ id: 'a1' });
      await CollaborationApi.getActivity('a1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/activities/a1');
    });
  });

  describe('getWatchers', () => {
    it('should get watchers', async () => {
      mockGet.mockResolvedValue([]);
      await CollaborationApi.getWatchers(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/watchers');
    });
  });

  describe('addWatcher', () => {
    it('should add watcher', async () => {
      mockPost.mockResolvedValue({ id: 'w1' });
      await CollaborationApi.addWatcher({ ticketId: 1, userId: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/watchers', { userId: 5 });
    });
  });

  describe('removeWatcher', () => {
    it('should remove watcher', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.removeWatcher(1, 'w1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/watchers/w1');
    });
  });

  describe('watchTicket', () => {
    it('should watch ticket', async () => {
      mockPost.mockResolvedValue({ id: 'w1' });
      await CollaborationApi.watchTicket(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/watch');
    });
  });

  describe('unwatchTicket', () => {
    it('should unwatch ticket', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.unwatchTicket(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/1/watch');
    });
  });

  describe('getCollaborationStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalComments: 10 });
      await CollaborationApi.getCollaborationStats(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/collaboration-stats');
    });
  });

  describe('getOnlinePresence', () => {
    it('should get presence', async () => {
      mockPost.mockResolvedValue([]);
      await CollaborationApi.getOnlinePresence([1, 2]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/presence/batch', { userIds: [1, 2] });
    });
  });

  describe('updatePresence', () => {
    it('should update presence', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.updatePresence({ status: 'online' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/presence/update', { status: 'online' });
    });
  });

  describe('sendTypingIndicator', () => {
    it('should send typing', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.sendTypingIndicator(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/1/typing');
    });
  });

  describe('getTypingUsers', () => {
    it('should get typing users', async () => {
      mockGet.mockResolvedValue([]);
      await CollaborationApi.getTypingUsers(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/typing');
    });
  });

  describe('deleteAttachment', () => {
    it('should delete attachment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await CollaborationApi.deleteAttachment('a1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/attachments/a1');
    });
  });

  describe('batchDeleteComments', () => {
    it('should batch delete comments', async () => {
      mockRequest.mockResolvedValue({ deleted: 2, failed: 0 });
      const result = await CollaborationApi.batchDeleteComments(['c1', 'c2']);
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/comments/batch', data: { commentIds: ['c1', 'c2'] } });
      expect(result.deleted).toBe(2);
    });
  });

  describe('batchMarkNotificationsAsRead', () => {
    it('should batch mark as read', async () => {
      mockPost.mockResolvedValue(undefined);
      await CollaborationApi.batchMarkNotificationsAsRead(['n1', 'n2']);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notifications/batch-read', { notificationIds: ['n1', 'n2'] });
    });
  });

  describe('searchComments', () => {
    it('should search comments', async () => {
      mockGet.mockResolvedValue({ comments: [], total: 0 });
      await CollaborationApi.searchComments({ query: 'test' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/comments/search', { query: 'test' });
    });
  });

  describe('exportComments', () => {
    it('should export comments', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      const result = await CollaborationApi.exportComments({ ticketId: 1, format: 'pdf' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/tickets/1/comments/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(CollaborationApi.getComment('999')).rejects.toThrow('Not found');
    });
  });
});
