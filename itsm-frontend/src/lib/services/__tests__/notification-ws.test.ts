/**
 * Notification WebSocket Service 测试
 */

// Mock the env module
jest.mock('@/lib/env', () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    warn: jest.fn(),
  },
}));

import {
  NotificationWSService,
  notificationWS,
  NotificationWSMessage,
} from '../notification-ws';
import { TicketNotification } from '@/lib/api/ticket-notification-api';

describe('NotificationWSService', () => {
  let service: NotificationWSService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new NotificationWSService();
  });

  afterEach(() => {
    service.disconnect();
  });

  describe('onNotification', () => {
    it('should register notification callback', () => {
      const callback = jest.fn();
      const unsubscribe = service.onNotification(callback);

      expect(typeof unsubscribe).toBe('function');
      unsubscribe();
    });

    it('should receive notifications', () => {
      const callback = jest.fn();
      service.onNotification(callback);

      const notification: TicketNotification = {
        id: 1,
        userId: 1,
        tenantId: 1,
        type: 'ticket_created',
        title: 'Test',
        content: 'Test content',
        isRead: false,
        createdAt: '2024-01-01T00:00:00Z',
      };

      const message: NotificationWSMessage = {
        type: 'notification',
        data: notification,
      };

      // Access private method for testing
      (service as unknown as { handleMessage: (msg: NotificationWSMessage) => void }).handleMessage(message);

      expect(callback).toHaveBeenCalledWith(notification);
    });
  });

  describe('onConnectionChange', () => {
    it('should register connection change callback', () => {
      const callback = jest.fn();
      const unsubscribe = service.onConnectionChange(callback);

      expect(typeof unsubscribe).toBe('function');
      unsubscribe();
    });
  });

  describe('disconnect', () => {
    it('should disconnect WebSocket', () => {
      service.disconnect();
      expect(service.isConnected()).toBe(false);
    });

    it('should reset state', () => {
      service.disconnect();
      // After disconnect, should be able to reconnect
      expect(service.isConnected()).toBe(false);
    });
  });

  describe('isConnected', () => {
    it('should return false when not connected', () => {
      expect(service.isConnected()).toBe(false);
    });
  });
});

describe('notificationWS singleton', () => {
  it('should export a singleton instance', () => {
    expect(notificationWS).toBeInstanceOf(NotificationWSService);
  });
});
