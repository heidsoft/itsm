import { TicketNotificationApi } from '@/lib/api/ticket-notification-api';
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

describe('TicketNotificationApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTicketNotifications', () => {
    it('should get ticket notifications', async () => {
      mockGet.mockResolvedValue({ notifications: [{ id: 1 }], total: 1 });
      const result = await TicketNotificationApi.getTicketNotifications(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/10/notifications');
      expect(result.notifications).toHaveLength(1);
    });
  });

  describe('sendTicketNotification', () => {
    it('should send notification', async () => {
      const data = { userIds: [1, 2], type: 'assigned', channel: 'email' as const, content: 'Test' };
      mockPost.mockResolvedValue(undefined);
      await TicketNotificationApi.sendTicketNotification(10, data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/10/notifications', data);
    });
  });

  describe('getUserNotifications', () => {
    it('should get user notifications', async () => {
      mockGet.mockResolvedValue({ notifications: [], total: 0, page: 1, pageSize: 10 });
      const result = await TicketNotificationApi.getUserNotifications({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notifications', { page: 1 });
      expect(result.total).toBe(0);
    });
  });

  describe('markNotificationRead', () => {
    it('should mark notification as read', async () => {
      mockPut.mockResolvedValue(undefined);
      await TicketNotificationApi.markNotificationRead(5);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notifications/5/read', {});
    });
  });

  describe('markAllNotificationsRead', () => {
    it('should mark all as read', async () => {
      mockPut.mockResolvedValue(undefined);
      await TicketNotificationApi.markAllNotificationsRead();
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notifications/read-all', {});
    });
  });

  describe('deleteNotification', () => {
    it('should delete a notification', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketNotificationApi.deleteNotification(5);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/notifications/5');
    });
  });

  describe('createNotification', () => {
    it('should create a notification', async () => {
      const payload = { title: 'Alert', content: 'Something happened' };
      mockPost.mockResolvedValue({ id: 1 });
      await TicketNotificationApi.createNotification(payload);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notifications', payload);
    });
  });

  describe('getNotificationPreferences', () => {
    it('should get preferences', async () => {
      mockGet.mockResolvedValue({ preferences: [], eventTypes: [] });
      const result = await TicketNotificationApi.getNotificationPreferences();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-preferences');
      expect(result.preferences).toEqual([]);
    });
  });

  describe('updateNotificationPreferences', () => {
    it('should update preferences', async () => {
      const data = { preferences: [{ eventType: 'created', emailEnabled: true, inAppEnabled: true, smsEnabled: false }] };
      mockPut.mockResolvedValue({ preferences: data.preferences, eventTypes: [] });
      await TicketNotificationApi.updateNotificationPreferences(data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notification-preferences', data);
    });
  });

  describe('resetNotificationPreferences', () => {
    it('should reset preferences', async () => {
      mockPost.mockResolvedValue({ reset: true });
      const result = await TicketNotificationApi.resetNotificationPreferences();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notification-preferences/reset', {});
      expect(result.reset).toBe(true);
    });
  });

  describe('initNotificationPreferences', () => {
    it('should init preferences', async () => {
      mockPost.mockResolvedValue({ initialized: true });
      const result = await TicketNotificationApi.initNotificationPreferences();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notification-preferences/init', {});
      expect(result.initialized).toBe(true);
    });
  });
});
