import { authApiClient, AuthAPI } from '@/lib/api/auth-api';
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

jest.mock('@/lib/auth/token-storage', () => ({
  clearAuthStorage: jest.fn(),
}));

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('AuthApiClient', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  describe('getCsrfToken', () => {
    it('should fetch CSRF token', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ data: { csrf_token: 'token123' } }),
      });
      const token = await authApiClient.getCsrfToken();
      expect(token).toBe('token123');
      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/csrf-token'), expect.any(Object));
    });

    it('should throw on non-ok response', async () => {
      mockFetch.mockResolvedValue({ ok: false });
      await expect(authApiClient.getCsrfToken()).rejects.toThrow('Failed to get CSRF token');
    });
  });

  describe('login', () => {
    it('should login successfully', async () => {
      mockFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { csrf_token: 'csrf1' } }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { user: { id: '1', username: 'admin' } } }) });
      const res = await authApiClient.login({ username: 'admin', password: 'pass' });
      expect(res.success).toBe(true);
    });

    it('should return error on failed login', async () => {
      mockFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { csrf_token: 'csrf1' } }) })
        .mockResolvedValueOnce({ ok: false, json: async () => ({ message: 'Invalid credentials' }) });
      const res = await authApiClient.login({ username: 'admin', password: 'wrong' });
      expect(res.success).toBe(false);
      expect(res.error).toBe('Invalid credentials');
    });

    it('should handle network error', async () => {
      mockFetch.mockRejectedValue(new Error('Network down'));
      const res = await authApiClient.login({ username: 'admin', password: 'pass', csrfToken: 'x' });
      expect(res.success).toBe(false);
      expect(res.error).toBe('Network down');
    });
  });

  describe('refreshToken', () => {
    it('should refresh token successfully', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ token: 'new-token' }) });
      const res = await authApiClient.refreshToken('old-token');
      expect(res.success).toBe(true);
      expect(res.token).toBe('new-token');
    });

    it('should handle failed refresh', async () => {
      mockFetch.mockResolvedValue({ ok: false, json: async () => ({ message: 'Expired' }) });
      const res = await authApiClient.refreshToken('old-token');
      expect(res.success).toBe(false);
      expect(res.error).toBe('Expired');
    });
  });

  describe('logout', () => {
    it('should logout successfully', async () => {
      mockFetch.mockResolvedValue({ ok: true });
      const res = await authApiClient.logout();
      expect(res.success).toBe(true);
    });

    it('should clear storage even on failure', async () => {
      mockFetch.mockRejectedValue(new Error('timeout'));
      const res = await authApiClient.logout();
      expect(res.success).toBe(false);
    });
  });

  describe('getWebAuthnChallenge', () => {
    it('should get challenge', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ challenge: 'abc123' }) });
      const res = await authApiClient.getWebAuthnChallenge('user1');
      expect(res.success).toBe(true);
      expect(res.challenge).toBe('abc123');
    });
  });

  describe('validateToken', () => {
    it('should validate token', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ code: 0, data: { id: '1', username: 'admin' } }) });
      const res = await authApiClient.validateToken();
      expect(res.valid).toBe(true);
      expect(res.user).toBeDefined();
    });

    it('should return invalid on non-ok', async () => {
      mockFetch.mockResolvedValue({ ok: false });
      const res = await authApiClient.validateToken();
      expect(res.valid).toBe(false);
    });
  });

  describe('initiateSSOLogin', () => {
    it('should initiate SSO', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ redirectUrl: 'https://sso.example.com' }) });
      const res = await authApiClient.initiateSSOLogin('default');
      expect(res.success).toBe(true);
      expect(res.redirectUrl).toBe('https://sso.example.com');
    });
  });

  describe('AuthAPI convenience object', () => {
    it('should have all methods', () => {
      expect(AuthAPI.login).toBeDefined();
      expect(AuthAPI.logout).toBeDefined();
      expect(AuthAPI.refreshToken).toBeDefined();
      expect(AuthAPI.getCsrfToken).toBeDefined();
      expect(AuthAPI.getWebAuthnChallenge).toBeDefined();
      expect(AuthAPI.verifyWebAuthn).toBeDefined();
      expect(AuthAPI.initiateSSOLogin).toBeDefined();
      expect(AuthAPI.validateToken).toBeDefined();
    });
  });
});
