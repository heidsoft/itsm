/**
 * Auth Service 测试
 */

import { AuthService } from '../auth-service';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('AuthService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getCurrentUser', () => {
    it('should return current user from store', () => {
      const user = AuthService.getCurrentUser();
      expect(user).toBeDefined();
    });
  });

  describe('isAuthenticated', () => {
    it('should return authentication status', () => {
      const result = AuthService.isAuthenticated();
      expect(typeof result).toBe('boolean');
    });
  });

  describe('getAccessToken', () => {
    it('should return access token', () => {
      const token = AuthService.getAccessToken();
      expect(token === null || typeof token === 'string').toBe(true);
    });
  });

  describe('getRefreshToken', () => {
    it('should return refresh token', () => {
      const token = AuthService.getRefreshToken();
      expect(token === null || typeof token === 'string').toBe(true);
    });
  });

  describe('getToken', () => {
    it('should return token (backward compatible)', () => {
      const token = AuthService.getToken();
      expect(token === null || typeof token === 'string').toBe(true);
    });
  });

  describe('setTokens', () => {
    it('should set tokens', () => {
      expect(() => AuthService.setTokens('access', 'refresh')).not.toThrow();
    });
  });

  describe('thirdPartyLogin', () => {
    it('should handle third party login', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'test-token', user: { id: 1 } }),
      });

      await expect(AuthService.thirdPartyLogin('google', 'code123')).resolves.not.toThrow();
    });

    it('should throw error on failed login', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
      });

      await expect(AuthService.thirdPartyLogin('google', 'invalid')).rejects.toThrow('登录失败');
    });
  });
});
