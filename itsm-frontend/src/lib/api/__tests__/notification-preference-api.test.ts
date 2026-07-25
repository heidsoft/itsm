import { NotificationPreferenceApi } from '@/lib/api/notification-preference-api';
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

describe('NotificationPreferenceApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getPreferences', () => {
    it('should get preferences', async () => {
      const expected = { preferences: [], eventTypes: ['ticket_created'] };
      mockGet.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.getPreferences();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-preferences');
      expect(res).toEqual(expected);
    });
  });

  describe('getMyPreferences', () => {
    it('should get my preferences', async () => {
      const expected = { id: 1, userId: 1, emailEnabled: true };
      mockGet.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.getMyPreferences();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-preferences');
      expect(res).toEqual(expected);
    });
  });

  describe('getPreferencesByUserId', () => {
    it('should get preferences by user id', async () => {
      const expected = { id: 1, userId: 5, emailEnabled: true };
      mockGet.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.getPreferencesByUserId(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-preferences/5');
      expect(res).toEqual(expected);
    });
  });

  describe('updateMyPreferences', () => {
    it('should update my preferences', async () => {
      const prefs = { emailEnabled: false, smsEnabled: true };
      const expected = { id: 1, userId: 1, ...prefs };
      mockPut.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.updateMyPreferences(prefs);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notification-preferences/me', prefs);
      expect(res).toEqual(expected);
    });
  });

  describe('updatePreferences', () => {
    it('should update preferences for user', async () => {
      const prefs = { emailEnabled: true };
      const expected = { id: 1, userId: 5, emailEnabled: true };
      mockPut.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.updatePreferences(5, prefs);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notification-preferences/5', prefs);
      expect(res).toEqual(expected);
    });
  });

  describe('resetToDefault', () => {
    it('should reset to default', async () => {
      const expected = { id: 1, userId: 1, emailEnabled: true };
      mockPost.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.resetToDefault();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notification-preferences/me/reset');
      expect(res).toEqual(expected);
    });
  });

  describe('getTemplates', () => {
    it('should get templates', async () => {
      const expected = [{ id: 1, userId: 0, emailEnabled: true }];
      mockGet.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.getTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-preferences/templates');
      expect(res).toEqual(expected);
    });
  });

  describe('applyTemplate', () => {
    it('should apply template', async () => {
      const expected = { id: 1, userId: 1, emailEnabled: true };
      mockPost.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.applyTemplate(3);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/notification-preferences/me/apply-template', { templateId: 3 });
      expect(res).toEqual(expected);
    });
  });

  describe('bulkUpdate', () => {
    it('should bulk update preferences', async () => {
      const data = { preferences: [{ eventType: 'ticket_created', emailEnabled: true, inAppEnabled: true }] };
      const expected = { preferences: [{}] };
      mockPut.mockResolvedValue(expected);
      const res = await NotificationPreferenceApi.bulkUpdate(data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/notification-preferences', data);
      expect(res).toEqual(expected);
    });
  });
});
